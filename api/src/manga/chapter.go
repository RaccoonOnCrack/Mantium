package manga

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diogovalentte/mantium/api/src/util"
)

type (
	// Type is the type of the chapter, it can be:
	// 0: "upload" - the chapter was uploaded, it's representing a chapter that was uploaded to (scraped from) a source
	// 1: "read" - the chapter was read, it's representing a chapter that was read by the user
	Type int
)

// Chapter is the struct for a chapter.
// Chapter don't has exported methods because a chapter should be used only by a manga.
type Chapter struct {
	// URL is the URL of the chapter
	URL string
	// Chapter usually is the chapter number, but in some cases it can be a one-shot or a special chapter
	Chapter string
	// Name is the name of the chapter
	Name string
	// UpdatedAt is the time when the chapter was uploaded or updated (read).
	// Should truncate at the second.
	// The timezone should be the default/system timezone.
	UpdatedAt time.Time
	Type      Type
}

func (c Chapter) String() string {
	return fmt.Sprintf("Chapter{URL: %s, Chapter: %s, Name: %s, UpdatedAt: %s, Type: %d}", c.URL, c.Chapter, c.Name, c.UpdatedAt, c.Type)
}

func insertChapterDB(c *Chapter, mangaID ID, tx *sql.Tx) (int, error) {
	contextError := "Error inserting chapter of manga ID '%d' in the database"

	err := validateChapter(c)
	if err != nil {
		return -1, util.AddErrorContext(err, fmt.Sprintf(contextError, mangaID))
	}
	var chapterID int
	err = tx.QueryRow(`
        INSERT INTO chapters
            (manga_id, url, chapter, name, updated_at, type)
        VALUES
            ($1, $2, $3, $4, $5, $6)
        RETURNING
            id;
    `, mangaID, c.URL, c.Chapter, c.Name, c.UpdatedAt, c.Type).Scan(&chapterID)
	if err != nil {
		return -1, util.AddErrorContext(err, fmt.Sprintf(contextError, mangaID))
	}

	return chapterID, nil
}

func getChapterDB(id int, db *sql.DB) (*Chapter, error) {
	contextError := "Error getting chapter with ID '%d' from the database"

	var chapter Chapter
	err := db.QueryRow(`
        SELECT
            url, chapter, name, updated_at, type
        FROM
            chapters
        WHERE
            id = $1;
    `, id).Scan(&chapter.URL, &chapter.Chapter, &chapter.Name, &chapter.UpdatedAt, &chapter.Type)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.AddErrorContext(fmt.Errorf("Chapter not found, is the ID correct?"), fmt.Sprintf(contextError, id))
		}
		return nil, util.AddErrorContext(err, fmt.Sprintf(contextError, id))
	}

	err = validateChapter(&chapter)
	if err != nil {
		return nil, util.AddErrorContext(err, fmt.Sprintf(contextError, id))
	}

	return &chapter, nil
}

// upsertMangaChapter updates the last upload or last read chapter of a manga
// if the manga doesn't exist in the database, it will be inserted
func upsertMangaChapter(m *Manga, chapter *Chapter, tx *sql.Tx) error {
	contextError := "Error upserting manga chapter in the database"

	err := validateManga(m)
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	err = validateChapter(chapter)
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	mangaID := m.ID
	if mangaID == 0 {
		mangaID, err = getMangaIDByURL(m.URL)
		if err != nil {
			return util.AddErrorContext(err, contextError)
		}
		m.ID = mangaID
	}

	var chapterID int
	err = tx.QueryRow(`
        INSERT INTO chapters (manga_id, url, chapter, name, updated_at, type)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT ON CONSTRAINT chapters_manga_id_type_unique
        DO UPDATE
            SET url = EXCLUDED.url, chapter = EXCLUDED.chapter, name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
        RETURNING id;
    `, m.ID, chapter.URL, chapter.Chapter, chapter.Name, chapter.UpdatedAt, chapter.Type).Scan(&chapterID)
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}

	var query string
	if chapter.Type == 1 {
		query = `
            UPDATE mangas
            SET last_upload_chapter = $1
            WHERE id = $2;
        `
	} else {
		query = `
            UPDATE mangas
            SET last_read_chapter = $1
            WHERE id = $2;
        `
	}

	var result sql.Result
	result, err = tx.Exec(query, chapterID, m.ID)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return util.AddErrorContext(err, contextError)
	}
	if rowsAffected == 0 {
		return util.AddErrorContext(fmt.Errorf("manga not found in DB"), contextError)
	}

	return nil
}

// there is no deleteChapterDB because the chapter should
// not be deleted directly, it should be deleted when a
// manga is deleted because of DB constraints

// valdiateChapter should be used every time the API interacts with
// the mangas and chapter table in the database
func validateChapter(c *Chapter) error {
	contextError := "Error validating chapter"

	if c.URL == "" {
		return util.AddErrorContext(fmt.Errorf("Chapter URL is empty"), contextError)
	}
	if c.Chapter == "" {
		return util.AddErrorContext(fmt.Errorf("Chapter chapter is empty"), contextError)
	}
	if c.Name == "" {
		return util.AddErrorContext(fmt.Errorf("Chapter name is empty"), contextError)
	}
	if c.Type != 1 && c.Type != 2 {
		return util.AddErrorContext(fmt.Errorf("Chapter type should be 1 (last upload) or 2 (last read), instead it's %d", c.Type), contextError)
	}

	return nil
}
