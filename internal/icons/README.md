# mudora-clipboard Icons

Assets obtained from:

* mudora's item sprites, from its `internal/icons/assets`
  - https://github.com/alttpr-mudora/mudora
  - which in turn credits The Spriter's Resource, the EmoTracker-Service default ALttPR pack and the `z3randomizer` base ASM patch (see mudora's own `internal/icons/README.md`)
* The Zelda Dungeon wiki's A Link to the Past enemies page, for the enemy sprites
  - https://www.zeldadungeon.net/wiki/A_Link_to_the_Past_Enemies

The sprites are in-game *A Link to the Past* artwork and belong to Nintendo.

## Sizes

The files here are resized copies, not the originals. Each sprite was trimmed of transparent padding and scaled with nearest neighbour (so the pixels stay hard-edged) until its longest side was about the same as every other sprite of its kind: about 32 px for items and about 40 px for enemies. A whole-number scale was used where one landed within 10% of that size, and the exact ratio otherwise. The web UI draws them at their natural size in fixed boxes (items 36 px, enemies 44 px), so nothing is scaled in the browser. `icons_test.go` fails if a sprite is not the right size, so scale any new one to match.

## Notes

* The plain Copy Rod (item id 1) has no sprite, and the silver bow reuses the bow.
* Guard variants that look alike share one enemy sprite, and Rock Hoarder and Stalfos Head have none.
