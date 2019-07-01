package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"

	"io/ioutil"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
)

var LINZ_API_KEY = getEnv("LINZ_API_KEY", "")
var BASE_PATH = getEnv("LINZ_BASE_PATH", "/mapcache")
var NZ_FILE = getEnv("LINZ_BOUND_FILE", "nz.geojson")

var b, _ = ioutil.ReadFile(NZ_FILE)
var nzBounds, _ = geojson.UnmarshalFeatureCollection(b)

var req, hit, miss int

func tileHandler(w http.ResponseWriter, r *http.Request) {
	req++

	// extract x, y and z
	re := regexp.MustCompile(`(aerial|topo)/(\d+)/(\d+)/(\d+)\.png$`)
	submatch := re.FindStringSubmatch(r.URL.Path)

	if !(len(submatch) == 5) {
		log.Printf("%s %s %s %s\n", r.RemoteAddr, r.Method, "MISSINGPARAM", r.RequestURI)
		http.NotFound(w, r)
		return
	}

	z, zerr := strconv.Atoi(submatch[2])
	x, xerr := strconv.Atoi(submatch[3])
	y, yerr := strconv.Atoi(submatch[4])

	layer := submatch[1]
	layerid := "set=2"
	if layer == "topo" {
		layerid = "layer=2343"
	}

	if xerr != nil || yerr != nil || zerr != nil {
		log.Printf("%s %s %s %s\n", r.RemoteAddr, r.Method, "BADNUMBER", r.RequestURI)
		http.NotFound(w, r)
		return
	}

	if !isXYZInsidePolygon(x, y, z) {
		log.Printf("%s %s %s %s\n", r.RemoteAddr, r.Method, "OUTOFBOUND", r.RequestURI)
		http.NotFound(w, r)
		return
	}

	// build the file path
	imgbase := path.Join(BASE_PATH, layer, i2s(z), i2s(x))
	imgpath := path.Join(imgbase, fmt.Sprintf("%d.png", y))

	// if the tile exists, serve it

	if FileExists(imgpath) {
		// serve file
		hit++
		log.Printf("%s %s %s %s\n", r.RemoteAddr, r.Method, "OK/HIT", r.RequestURI)
		w.Header().Set("Content-Type", "image/png")
		http.ServeFile(w, r, imgpath)
		return
	}

	// have we logged a miss before?
	if FileExists(imgpath + ".404") {
		hit++
		log.Printf("%s %s %s %s\n", r.RemoteAddr, r.Method, "MISSING/HIT", r.RequestURI)
		http.NotFound(w, r)
		return
	}

	os.MkdirAll(imgbase, os.ModePerm)

	tileurl := fmt.Sprintf("https://tiles-a.data-cdn.linz.govt.nz/services;key=%s/tiles/v4/%s/EPSG:3857/%d/%d/%d.png",
		LINZ_API_KEY, layerid, z, x, y)

	// download and serve the tile
	if DownloadTile(tileurl, imgpath) {
		miss++
		log.Printf("%s %s %s %s\n", r.RemoteAddr, r.Method, "OK/MISS", r.RequestURI)
		w.Header().Set("Content-Type", "image/png")
		http.ServeFile(w, r, imgpath)
		return
	}

	log.Printf("%s %s %s %s\n", r.RemoteAddr, r.Method, "UNKNOWN", r.RequestURI)
	http.NotFound(w, r)
	return
}

func DownloadTile(url, path string) bool {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
		return false
	}

	req.Header.Set("User-Agent", "LINZ MapCache (for OpenStreetMap); cameron@dewitte.id.au")

	resp, err := client.Do(req)
	if err != nil {
		emptyFile, err := os.Create(path + ".404")
		if err != nil {
			log.Fatalln(err)
		} else {
			emptyFile.Close()
		}
		return false
	}

	if resp.StatusCode != http.StatusOK {
		return false
	}

	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		log.Fatalln(err)
		return false
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return true
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	}
	return false
}

func isXYZInsidePolygon(x, y, z int) bool {
	corners := [4]orb.Point{
		XYZ2LL(x, y, z),
		XYZ2LL(x+1, y, z),
		XYZ2LL(x, y+1, z),
		XYZ2LL(x+1, y+1, z),
	}

	for c := 0; c < 4; c++ {
		if isPointInsidePolygon(corners[c]) {
			return true
		}
	}
	return false
}

func isPointInsidePolygon(point orb.Point) bool {
	for _, feature := range nzBounds.Features {
		// Try on a MultiPolygon to begin
		multiPoly, isMulti := feature.Geometry.(orb.MultiPolygon)
		if isMulti {
			if planar.MultiPolygonContains(multiPoly, point) {
				return true
			}
		} else {
			// Fallback to Polygon
			polygon, isPoly := feature.Geometry.(orb.Polygon)
			if isPoly {
				if planar.PolygonContains(polygon, point) {
					return true
				}
			}
		}
	}
	return false
}

func XYZ2LL(x, y, z int) orb.Point {
	xf := float64(x)
	yf := float64(y)
	zf := float64(z)

	long := xf/math.Exp2(zf)*360.0 - 180.0
	n := math.Pi - 2*math.Pi*yf/math.Exp2(zf)
	lat := 180 / math.Pi * math.Atan(0.5*(math.Exp(n)-math.Exp(-n)))

	return orb.Point{long, lat}
}

func i2s(i int) string {
	return strconv.Itoa(i)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "{\"requests\": %d, \"hit\":%d, \"miss\": %d}", req, hit, miss)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	if LINZ_API_KEY == "" {
		log.Fatal("Provide an API key through the LINZ_API_KEY environment variable")
		return
	}

	http.HandleFunc("/linz_aerial/", tileHandler)
	http.HandleFunc("/linz_topo/", tileHandler)
	http.HandleFunc("/linz_stats/", statsHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))

}
