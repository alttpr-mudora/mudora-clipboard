// Enemy and item names for the Copy Rod hack. The item ids were worked out from one seed (YZoP82YAEr);
// ids 0x20-0x27 were never seen and are unnamed.

package copyrod

import "fmt"

// Enemy is a sprite type the Copy Rod hack can assign an item to.
type Enemy struct {
	Type byte
	Name string
}

// Enemies lists the 78 enemy sprite types in the order the hack stores them ($BA:8E41).
var Enemies = []Enemy{
	{0x00, "Raven"},
	{0x01, "Vulture"},
	{0x08, "Octorok"},
	{0x0A, "4-Way Octorok"},
	{0x0B, "Cucco"},
	{0x0D, "Buzzblob"},
	{0x0E, "Snapdragon"},
	{0x0F, "Octoballoon"},
	{0x11, "Hinox"},
	{0x12, "Moblin"},
	{0x13, "Mini Helmasaur"},
	{0x15, "Antifairy"},
	{0x17, "Bush Hoarder"},
	{0x18, "Mini Moldorm"},
	{0x19, "Poe"},
	{0x20, "Sluggula"},
	{0x22, "Ropa"},
	{0x23, "Red Bari"},
	{0x24, "Blue Bari"},
	{0x26, "Hardhat Beetle"},
	{0x27, "Deadrock"},
	{0x3E, "Rock Hoarder"},
	{0x41, "Blue Guard"},
	{0x42, "Green Guard"},
	{0x43, "Red Spear Guard"},
	{0x44, "Fast Blue Guard"},
	{0x45, "Fast Red Guard / Summoned Guard"},
	{0x46, "Blue Archer"},
	{0x47, "Green Bush Guard"},
	{0x48, "Red Javelin Guard"},
	{0x49, "Red Bush Guard"},
	{0x4A, "Bomb Guard"},
	{0x4B, "Green Knife Guard"},
	{0x4C, "Geldman"},
	{0x4D, "Toppo"},
	{0x4E, "Popo"},
	{0x51, "Armos Statue"},
	{0x55, "Shooting Zora"},
	{0x56, "Walking Zora"},
	{0x58, "Crab"},
	{0x64, "Debirando"},
	{0x6A, "Ball-and-Chain Guard"},
	{0x6D, "Rat / Cricket"},
	{0x6E, "Snake"},
	{0x6F, "Keese"},
	{0x71, "Leever"},
	{0x79, "Bee"},
	{0x7C, "Stalfos Head"},
	{0x81, "Hover"},
	{0x83, "Green Eyegore / Mimic"},
	{0x84, "Red Eyegore / Mimic"},
	{0x85, "Falling Stalfos"},
	{0x86, "Kodongo"},
	{0x8B, "Gibdo"},
	{0x8E, "Terrorpin"},
	{0x8F, "Zol / Blob"},
	{0x90, "Wallmaster"},
	{0x91, "Stalfos Knight"},
	{0x94, "Pirogusu"},
	{0x99, "Pengator"},
	{0x9A, "Kyameron"},
	{0x9B, "Wizzrobe"},
	{0x9D, "Babasu"},
	{0xA1, "Freezor"},
	{0xA5, "Blue Zazak"},
	{0xA6, "Red Zazak"},
	{0xA7, "Stalfos"},
	{0xA8, "Green Zirro"},
	{0xA9, "Blue Zirro"},
	{0xAA, "Pikit"},
	{0xC3, "Gibo"},
	{0xC4, "Thief"},
	{0xC7, "Hokkubokku"},
	{0xC9, "Tektite"},
	{0xCF, "Swamola"},
	{0xD0, "Lynel"},
	{0xD1, "Bunnybeam"},
	{0xD3, "Stal"},
}

// Items maps a copied-item id (the bytes in the enemy table) to its name.
// 0x00 means no item and 0x01 is the plain Copy Rod.
var Items = map[byte]string{
	0x01: "Copy Rod",
	0x02: "Bow",
	0x03: "Silver Bow",
	0x04: "Blue Boomerang",
	0x05: "Red Boomerang",
	0x06: "Hookshot",
	0x07: "Bombs",
	0x08: "Mushroom",
	0x09: "Powder",
	0x0A: "Fire Rod",
	0x0B: "Ice Rod",
	0x0C: "Bombos",
	0x0D: "Ether",
	0x0E: "Quake",
	0x0F: "Lamp",
	0x10: "Hammer",
	0x11: "Shovel",
	0x12: "Flute",
	0x13: "Bug Net",
	0x14: "Book of Mudora",
	0x15: "Cane of Somaria",
	0x16: "Cane of Byrna",
	0x17: "Cape",
	0x18: "Mirror",
	0x19: "Glove",
	0x1A: "Boots",
	0x1B: "Flippers",
	0x1C: "Moon Pearl",
	0x1D: "Sword",
	0x1E: "Armor",
	0x1F: "Magic",
}

// ItemName returns the display name of a copied-item id.
func ItemName(id byte) string {
	if n, ok := Items[id]; ok {
		return n
	}
	if id == 0 {
		return "(no item)"
	}
	return fmt.Sprintf("item 0x%02X", id)
}
