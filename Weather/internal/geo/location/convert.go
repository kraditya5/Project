package location

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/samonzeweb/godb"
	"github.com/samonzeweb/godb/adapters/sqlite"
)

// ConvertCache converts files-based cache into the DB-based cache.
// If reset is true, the DB cache is created from scratch.
//
//nolint:funlen,cyclop
func (c *Cache) ConvertCache(reset bool) error {
	var (
		dbfile     = c.config.Geo.LocationCacheDB
		tableName  = "Location"
		cacheFiles = c.filesCacheDir
		known      = map[string]bool{}
	)

	if reset {
		err := removeDBIfExists(dbfile)
		if err != nil {
			return err
		}
	}

	db, err := godb.Open(sqlite.Adapter, dbfile)
	if err != nil {
		return err
	}

	if reset {
		err = createTable(db, tableName)
		if err != nil {
			return err
		}
	}

	log.Println("listing cache entries...")
	files, err := filepath.Glob(filepath.Join(cacheFiles, "*"))
	if err != nil {
		return err
	}

	log.Printf("going to convert %d entries\n", len(files))

	block := []Location{}
	for i, file := range files {
		ip := filepath.Base(file)
		loc, err := c.Read(ip)
		if err != nil {
			log.Println("invalid entry for", ip)

			continue
		}

		// Skip too long location names.
		if len(loc.Name) > 25 {
			continue
		}

		// Skip duplicates.
		if known[loc.Name] {
			log.Println("skipping", loc.Name)

			continue
		}

		singleLocation := Location{}
		err = db.Select(&singleLocation).
			Where("name = ?", loc.Name).
			Do()
		if !errors.Is(err, sql.ErrNoRows) {
			log.Println("found in db:", loc.Name)

			continue
		}

		known[loc.Name] = true

		// Skip some invalid names.
		if strings.Contains(loc.Name, "\n") {
			continue
		}

		block = append(block, *loc)
		if i%1000 != 0 || i == 0 {
			continue
		}

		log.Println("going to insert new entries")
		err = db.BulkInsert(&block).Do()
		if err != nil {
			return err
		}
		block = []Location{}
		log.Println("converted", i+1, "entries")
	}

	// inserting the rest.
	err = db.BulkInsert(&block).Do()
	if err != nil {
		return err
	}

	log.Println("converted", len(files), "entries")

	return nil
}

func createTable(db *godb.DB, tableName string) error {
	createTable := fmt.Sprintf(
		`create table %s (
	    name           text not null primary key,
        displayName    text not null,
        lat            text not null,
        lon            text not null,
		timezone       text not null);
	`, tableName)

	_, err := db.CurrentDB().Exec(createTable)

	return err
}

func removeDBIfExists(filename string) error {
	_, err := os.Stat(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// no db file
		return nil
	}

	return os.Remove(filename)
}
