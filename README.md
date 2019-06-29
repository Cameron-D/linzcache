There are already many tools and servers for caching map tiles but nothing that worked just how I wanted, and I've also wanted to try using Go for a while, so this came together in a weekend (plus a bit, cleaning up and fixing the code).

Simply, it listens for ZXY reqests within a set boundary, checks if a copy exists on disk, and if not it will query the tile from LINZ/Koordinates and save the response for later requests.

Because it stores files on disk there is no cache expiration ability, just delete the files from disk. I'll soon have a script to purge tiles by region to deal with source data updates.

### In operation
* `https://map.cazzaserver.com/linz_aerial/{zoom}/{x}/{y}.png` for aerial images. [Slippy map preview](https://map.cazzaserver.com/linz_aerial.html#map=8/19323065.31/-5162122.28/0).
* `https://map.cazzaserver.com/linz_topo/{zoom}/{x}/{y}.png` for topo maps, [Slippy map preview](https://map.cazzaserver.com/linz_topo.html#map=8/19323065.31/-5162122.28/0).