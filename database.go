package main

import (
	"log"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

type staticCommand struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
	Call      string
	Message   string
	LastUseBy string
	LastUse   time.Time
}

func (b *bot) newDatabase() {
	var err error
	b.db, err = gorm.Open("sqlite3", b.config.Database)
	if err != nil {
		log.Fatalf("failed opening database: %s", err)
	}
	b.db.AutoMigrate(&staticCommand{})
}

// AddStaticCommand loads all commands stored in the database
func (b *bot) LoadStaticCommands() {
	b.db.Find(&b.commands)
}

// AddCommand adds the command to the database or updates it
func (b *bot) AddStaticCommand(c *staticCommand, user string) (*staticCommand, error) {
	// create a new entry if it doesn't exist
	if c.ID == 0 {
		b.db.Create(&c)
		return c, b.db.Error
	}
	b.db.Save(&c)
	return c, b.db.Error
}

// UpdateStaticCommand updates the given command
func (b *bot) UpdateStaticCommand(c *staticCommand) error {
	b.db.Save(&c)
	return b.db.Error
}

// DeleteStaticCommands delete the given command
func (b *bot) DeleteStaticCommands(c *staticCommand) error {
	b.db.Delete(&c)
	return b.db.Error
}
