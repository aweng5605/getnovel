package bs

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/missdeer/golib/httputil"
)

// Book represent a book instance
type Book struct {
	Tag          string        `json:"tag"`
	Origin       string        `json:"origin"`
	Name         string        `json:"name"`
	Author       string        `json:"author"`
	BookmarkList []interface{} `json:"bookmarkList"`
	ChapterURL   string        `json:"chapterUrl"`
	// BookURL        string            `json:"book_url"`
	CoverURL         string            `json:"coverUrl"`
	Kind             string            `json:"kind"`
	LastChapter      string            `json:"lastChapter"`
	FinalRefreshData UnixTime          `json:"finalRefreshData"`
	NoteURL          string            `json:"noteUrl"`
	Introduce        string            `json:"introduce"`
	ChapterList      []*Chapter        `json:"-"`
	BookSourceInst   *BookSource       `json:"-"`
	Page             *goquery.Document `json:"-"`
}

func (b Book) String() string {
	return fmt.Sprintf("%s( %s )", b.Name, b.NoteURL)
}

// FindBookSourceForBook find book source for a specified book
func FindBookSourceForBook(b *Book) *BookSource {
	if b.BookSourceInst != nil {
		return b.BookSourceInst
	}
	if b.Tag == "" {
		if b.NoteURL == "" {
			return nil
		}
		b.Tag = httputil.GetHostByURL(b.NoteURL)
	}
	if bsItem, ok := bsCache.Get(b.Tag); ok {
		if bs, ok := bsItem.(BookSource); ok {
			b.BookSourceInst = &bs
			b.Origin = bs.BookSourceName
			return &bs
		} else {
			return nil
		}
	}
	return nil
}

// NewBookFromURL create new Book instance from URL
func NewBookFromURL(bookURL string) (*Book, error) {
	if bookURL == "" {
		return nil, errors.New("no url.")
	}
	_, err := url.ParseRequestURI(bookURL)
	if err != nil {
		return nil, err
	}
	b := &Book{
		NoteURL: bookURL,
		Tag:     httputil.GetHostByURL(bookURL),
	}
	b.GetAuthor()
	b.GetIntroduce()
	return b, nil
}

func (b *Book) getBookPage() (*goquery.Document, error) {
	if b.Page != nil {
		return b.Page, nil
	}
	bs := FindBookSourceForBook(b)
	if b.NoteURL != "" && bs != nil {
		p, err := httputil.GetPage(b.NoteURL, FindBookSourceForBook(b).HTTPUserAgent)
		if err == nil {
			doc, err := goquery.NewDocumentFromReader(p)
			if err == nil {
				b.Page = doc
				return b.Page, nil
			}
		}
		return nil, err
	}
	return nil, errors.New("can't get book page.")
}

func (b *Book) GetChapterURL() string {
	if b.ChapterURL != "" {
		return b.ChapterURL
	}
	doc, err := b.getBookPage()
	if err == nil {
		_, chapterURL := ParseRules(doc, b.BookSourceInst.RuleChapterURL)
		if chapterURL != "" {
			log.Printf("chapter url is: %s", chapterURL)
			b.ChapterURL = chapterURL
			return b.ChapterURL
		}
	} else {
		log.Printf("get chapterURL error:%s\n", err.Error())
	}
	return b.NoteURL
}

func (b *Book) GetChapterList() []*Chapter {
	b.UpdateChapterList(len(b.ChapterList))
	return b.ChapterList
}

func (b *Book) UpdateChapterList(startFrom int) error {
	var doc *goquery.Document
	var err error
	bs := FindBookSourceForBook(b)
	// if b.ChapterURL != "" && bs != nil {
	p, err := httputil.GetPage(b.GetChapterURL(), FindBookSourceForBook(b).HTTPUserAgent)
	log.Printf("%s chapterlist url is:%s .", b.Name, b.ChapterURL)
	if err != nil {
		log.Printf("error while getting chapter list page: %s", err.Error())
	}
	doc, err = goquery.NewDocumentFromReader(p)
	if err != nil {
		log.Printf("error while parsing chapter list page to goquery: %s", err.Error())
	}
	// }
	if doc == nil {
		log.Printf("%s no chapterurl found.got by bookurl.", bs.BookSourceName)
		doc, err = b.getBookPage()
		if err != nil {
			return err
		}
	}
	sel, _ := ParseRules(doc, b.BookSourceInst.RuleChapterList)
	if sel != nil {
		sel.Each(func(i int, s *goquery.Selection) {
			if i < startFrom {
				return
			}
			_, name := ParseRules(s, b.BookSourceInst.RuleChapterName)
			_, url := ParseRules(s, b.BookSourceInst.RuleContentURL)
			if strings.HasPrefix(url, "/") {
				url = fmt.Sprintf("%s%s", b.BookSourceInst.BookSourceURL, url)
			}
			b.ChapterList = append(b.ChapterList, &Chapter{
				ChapterTitle: name,
				ChapterURL:   url,
				BelongToBook: b,
				Index:        i,
			})
		})
	}
	return nil
}

func (b *Book) GetName() string {
	if b.Name != "" {
		return b.Name
	}
	doc, err := b.getBookPage()
	if err == nil {
		_, title := ParseRules(doc, b.BookSourceInst.RuleBookName)
		if title != "" {
			b.Name = title
		}
	} else {
		log.Printf("get title error:%s\n", err.Error())
	}
	return b.Name
}

func (b *Book) GetIntroduce() string {
	if b.Introduce != "" {
		return b.Introduce
	}
	doc, err := b.getBookPage()
	if err == nil {
		_, intro := ParseRules(doc, b.BookSourceInst.RuleIntroduce)
		if intro != "" {
			b.Introduce = intro
		}
	} else {
		log.Printf("get introduce error:%s\n", err.Error())
	}
	return b.Introduce
}

func (b *Book) GetAuthor() string {
	if b.Author == "" {
		doc, err := b.getBookPage()
		if err == nil {
			_, intro := ParseRules(doc, b.BookSourceInst.RuleBookAuthor)
			if intro != "" {
				b.Author = intro
			}
		} else {
			log.Printf("get author error:%s\n", err.Error())
		}
	}
	return b.Author
}
