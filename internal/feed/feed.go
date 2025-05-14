package feed

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/45uperman/gator/internal/database"
	"github.com/google/uuid"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

func (rf RSSFeed) Unescape() {
	rf.Channel.Title = html.UnescapeString(rf.Channel.Title)
	rf.Channel.Description = html.UnescapeString(rf.Channel.Description)

	for i := range rf.Channel.Item {
		rf.Channel.Item[i].Title = html.UnescapeString(rf.Channel.Item[i].Title)
		rf.Channel.Item[i].Description = html.UnescapeString(rf.Channel.Item[i].Description)
	}
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	newFeed := &RSSFeed{}
	err = xml.Unmarshal(data, newFeed)
	if err != nil {
		return nil, err
	}

	return newFeed, nil
}

func ScrapeFeeds(db *database.Queries) error {
	nextFeed, err := db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	err = db.MarkFeedFetched(
		context.Background(),
		database.MarkFeedFetchedParams{
			LastFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
			ID:            nextFeed.ID,
		},
	)
	if err != nil {
		return err
	}

	rssFeed, err := fetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		return err
	}
	rssFeed.Unescape()

	for _, item := range rssFeed.Channel.Item {
		postTitle := sql.NullString{String: item.Title, Valid: true}
		if postTitle.String == "" {
			postTitle.Valid = false
		}

		postDescription := sql.NullString{String: item.Description, Valid: true}
		if postDescription.String == "" {
			postDescription.Valid = false
		}

		pubTime, err := parsePubDate(item.PubDate)
		if err != nil {
			fmt.Printf("error saving post '%s' from feed '%s': %s\n", item.Title, nextFeed.Name, err)
		}
		postPubTime := sql.NullTime{Time: pubTime, Valid: false}
		if err == nil {
			postPubTime.Valid = true
		}

		post, err := db.CreatePost(
			context.Background(),
			database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Title:       sql.NullString{String: item.Title, Valid: true},
				Url:         item.Link,
				Description: sql.NullString{String: item.Description, Valid: true},
				PublishedAt: postPubTime,
				FeedID:      nextFeed.ID,
			},
		)
		if err != nil {
			if err.Error() != `pq: duplicate key value violates unique constraint "posts_url_key"` {
				fmt.Println(err)
			}
			continue
		}

		fmt.Printf("Saved post '%s' from feed '%s'\n", post.Title.String, nextFeed.Name)
	}

	return nil
}

func parsePubDate(pubDate string) (time.Time, error) {
	if pubDate == "" {
		return time.Time{}, fmt.Errorf("post has no publish date")
	}

	knownFormats := []string{
		time.Layout,
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range knownFormats {
		pubTime, err := time.Parse(format, pubDate)
		if err == nil {
			return pubTime, nil
		}
	}

	return time.Time{}, fmt.Errorf("unknown publish date format")
}
