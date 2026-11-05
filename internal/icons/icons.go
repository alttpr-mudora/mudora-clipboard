// Package icons holds the sprites of the web UI, embedded in the binary: an icon for every item the Copy Rod hack
// can hand out and a sprite for (nearly) every enemy it can attach one to.
//
// Every sprite is scaled ahead of time so that all item icons are one size and all enemy icons another (see
// README.md), and the page shows them at their natural size.
package icons

import (
	"embed"
	"io/fs"
)

//go:embed assets
var assets embed.FS

// FS returns the sprites as a file system rooted at the assets folder, so files are "items/<name>" and
// "enemies/<name>". The names Item and Enemy return are paths in it.
func FS() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err) // the embedded folder is part of the build
	}
	return sub
}

// Item returns the path of the sprite for a copied-item id, or "" if it has none. The plain Copy Rod (id 1) has
// none, and the silver bow reuses the bow.
func Item(id byte) string {
	if f, ok := itemFiles[id]; ok {
		return "items/" + f
	}
	return ""
}

// Enemy returns the path of the sprite for an enemy sprite type, or "" if it has none. Guard variants that look
// alike share a file. Two enemies have none: Rock Hoarder (0x3E, the source's Hoarder is the bush one) and
// Stalfos Head (0x7C).
func Enemy(t byte) string {
	if f, ok := enemyFiles[t]; ok {
		return "enemies/" + f
	}
	return ""
}

// Item files, by copied-item id. They follow mudora's internal/icons/icons.go, which maps by item name.
var itemFiles = map[byte]string{
	0x02: "bow.png",
	0x03: "bow.png", // mudora has no silver bow sprite
	0x04: "boomerang-blue.png",
	0x05: "boomerang-red.png",
	0x06: "hookshot.png",
	0x07: "bomb-1.png",
	0x08: "mushroom.png",
	0x09: "powder.png",
	0x0A: "fire-rod.png",
	0x0B: "ice-rod.png",
	0x0C: "bombos.png",
	0x0D: "ether.png",
	0x0E: "quake.png",
	0x0F: "lamp.png",
	0x10: "hammer.png",
	0x11: "shovel.png",
	0x12: "flute.png",
	0x13: "bugnet.png",
	0x14: "book.png",
	0x15: "somaria.png",
	0x16: "byrna.png",
	0x17: "cape.png",
	0x18: "magic-mirror.png",
	0x19: "glove.png",
	0x1A: "boots.png",
	0x1B: "flippers.png",
	0x1C: "moonpearl.png",
	0x1D: "sword.png",
	0x1E: "mail-green.png",
	0x1F: "magic-large.png",
}

// Enemy files, by enemy sprite type; the comment on each is the enemy.
var enemyFiles = map[byte]string{
	0x00: "Crow.png",                       // Raven
	0x01: "Tendoru-Sprite-1.png",           // Vulture
	0x08: "Octorok-Sprite-3.png",           // Octorok
	0x0A: "Orange-Octorok.png",             // 4-Way Octorok
	0x0B: "Cucco-Sprite.png",               // Cucco
	0x0D: "Buzz-Blob-2.png",                // Buzzblob
	0x0E: "Snap-Dragon-2.png",              // Snapdragon
	0x0F: "Octoballoon-Sprite-1.png",       // Octoballoon
	0x11: "Hinox-Sprite-1.png",             // Hinox
	0x12: "Moblin-Sprite-1.png",            // Moblin
	0x13: "Helmasaur-1.png",                // Mini Helmasaur
	0x15: "AntiFairy.png",                  // Antifairy
	0x17: "Hoarder-Sprite-1.png",           // Bush Hoarder
	0x18: "Mini-Moldorm-1.png",             // Mini Moldorm
	0x19: "Poe-Sprite-1.png",               // Poe
	0x20: "Nuranuru-2.png",                 // Sluggula
	0x22: "Ropa-Sprite-2.png",              // Ropa
	0x23: "Red-Bari-1.png",                 // Red Bari
	0x24: "Blue-Bari-1.png",                // Blue Bari
	0x26: "Blue-Hardhat-Beetle-1.png",      // Hardhat Beetle
	0x27: "Deadrock-Sprite-2.png",          // Deadrock
	0x41: "Blue-Sword-Soldier-Sprite.png",  // Blue Guard
	0x42: "Soldier-Sprite.png",             // Green Guard
	0x43: "Red-Spear-Soldier-Sprite-1.png", // Red Spear Guard
	0x44: "Blue-Sword-Soldier-Sprite.png",  // Fast Blue Guard
	0x45: "Red-Spear-Soldier-Sprite-1.png", // Fast Red Guard / Summoned Guard
	0x46: "Archer.png",                     // Blue Archer
	0x47: "Green-Sword-Soldier-Sprite.png", // Green Bush Guard
	0x48: "Spear-Soldier-Sprite-1.png",     // Red Javelin Guard
	0x49: "Red-Spear-Soldier-Sprite-1.png", // Red Bush Guard
	0x4A: "Bomb-Knight-Sprite.png",         // Bomb Guard
	0x4B: "Green-Sword-Soldier-Sprite.png", // Green Knife Guard
	0x4C: "Geldman-Sprite.png",             // Geldman
	0x4D: "Toppo-Sprite-1.png",             // Toppo
	0x4E: "Popo-1.png",                     // Popo
	0x51: "Armos-Sprite-1.png",             // Armos Statue
	0x55: "Zora-Sprite-1.png",              // Shooting Zora
	0x56: "Zora-Sprite-3.png",              // Walking Zora
	0x58: "Sand-Crab-Sprite.png",           // Crab
	0x64: "Blue-Debirando-1.png",           // Debirando
	0x6A: "BallAndChainTrooper.png",        // Ball-and-Chain Guard
	0x6D: "Rat-Sprite-3.png",               // Rat / Cricket
	0x6E: "Rope-Sprite-1.png",              // Snake
	0x6F: "Keese-Sprite.png",               // Keese
	0x71: "Leever-Green-1.png",             // Leever
	0x79: "Bee-2.png",                      // Bee
	0x81: "Hover-1.png",                    // Hover
	0x83: "Eyegore.png",                    // Green Eyegore / Mimic
	0x84: "Red-Eyegore-Sprite-2.png",       // Red Eyegore / Mimic
	0x85: "Stalfos-ALTTP-Blue-2.png",       // Falling Stalfos
	0x86: "Kodondo-Green-1.png",            // Kodongo
	0x8B: "Gibdo-1.png",                    // Gibdo
	0x8E: "Terrorpin.png",                  // Terrorpin
	0x8F: "Zol-Green-1.png",                // Zol / Blob
	0x90: "Wallmaster-2.png",               // Wallmaster
	0x91: "Stalfos-Knight-2.png",           // Stalfos Knight
	0x94: "AquaBeetle.png",                 // Pirogusu
	0x99: "Pengator-4.png",                 // Pengator
	0x9A: "Kyameron-2.png",                 // Kyameron
	0x9B: "Wizzrobe-Sprite-ALTTP.png",      // Wizzrobe
	0x9D: "Babusu-1.png",                   // Babasu
	0xA1: "Freezor.png",                    // Freezor
	0xA5: "Blue-Zazakku-1.png",             // Blue Zazak
	0xA6: "Red-Zazakku-2.png",              // Red Zazak
	0xA7: "Stalfos-ALTTP-Yellow-1.png",     // Stalfos
	0xA8: "Zirro-Green-1.png",              // Green Zirro
	0xA9: "Zirro-Purple-1.png",             // Blue Zirro
	0xAA: "Pikit-1.png",                    // Pikit
	0xC3: "Gibo-1.png",                     // Gibo
	0xC4: "Thief-Enemy.png",                // Thief
	0xC7: "Hokkubokku-2.png",               // Hokkubokku
	0xC9: "Blue-Tektite-2.png",             // Tektite
	0xCF: "Swamola-3.png",                  // Swamola
	0xD0: "Lynel-1.png",                    // Lynel
	0xD1: "Rabbit-Beam-2.png",              // Bunnybeam
	0xD3: "Stal-1.png",                     // Stal
}
