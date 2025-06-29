package htmlparser

import (
	"strings"
	"testing"
)

func TestDocument_Find(t *testing.T) {
	html := `
		<html>
			<body>
				<div class="container">
					<article class="elementor-post">
						<h5 class="elementor-post__title">
							<a href="https://example.com/post1">Post Title 1</a>
						</h5>
						<a class="elementor-post__thumbnail__link" href="https://example.com/post1">Thumbnail</a>
					</article>
					<article class="elementor-post">
						<h5 class="elementor-post__title">
							<a href="https://example.com/post2">Post Title 2</a>
						</h5>
					</article>
				</div>
				<div class="item-list">
					<div class="post-box-title">
						<a href="https://example.com/post3">Post Title 3</a>
					</div>
				</div>
				<div id="btn-download" data-link="https://download.com/file.zip">Download</div>
			</body>
		</html>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Test finding articles
	articles := doc.Find("article.elementor-post")
	if len(articles.nodes) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(articles.nodes))
	}

	// Test finding links within articles
	var titles []string
	var urls []string

	articles.Each(func(i int, article *Element) {
		link := article.Find("a.elementor-post__thumbnail__link, h5.elementor-post__title a").First()
		if url, exists := link.Attr("href"); exists {
			urls = append(urls, url)
		}

		titleEl := article.Find("h5.elementor-post__title")
		if len(titleEl.nodes) > 0 {
			title := strings.TrimSpace(titleEl.First().Text())
			if title != "" {
				titles = append(titles, title)
			}
		}
	})

	expectedTitles := []string{"Post Title 1", "Post Title 2"}
	expectedURLs := []string{"https://example.com/post1", "https://example.com/post2"}

	if len(titles) != len(expectedTitles) {
		t.Errorf("Expected %d titles, got %d", len(expectedTitles), len(titles))
	}

	for i, title := range titles {
		if i < len(expectedTitles) && title != expectedTitles[i] {
			t.Errorf("Expected title '%s', got '%s'", expectedTitles[i], title)
		}
	}

	if len(urls) != len(expectedURLs) {
		t.Errorf("Expected %d URLs, got %d", len(expectedURLs), len(urls))
	}

	for i, url := range urls {
		if i < len(expectedURLs) && url != expectedURLs[i] {
			t.Errorf("Expected URL '%s', got '%s'", expectedURLs[i], url)
		}
	}
}

func TestDocument_FindWithAttributes(t *testing.T) {
	html := `
		<html>
			<body>
				<div id="btn-download" data-link="https://download.com/file.zip">Download</div>
				<a data-e-disable-page-transition="true" href="https://example.com/download">Download Link</a>
			</body>
		</html>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Test finding by ID
	downloadBtn := doc.Find("#btn-download").First()
	if dataLink, exists := downloadBtn.Attr("data-link"); !exists || dataLink != "https://download.com/file.zip" {
		t.Errorf("Expected data-link='https://download.com/file.zip', got '%s' (exists: %v)", dataLink, exists)
	}

	// Test finding by attribute
	downloadLink := doc.Find("a[data-e-disable-page-transition=true]").First()
	if href, exists := downloadLink.Attr("href"); !exists || href != "https://example.com/download" {
		t.Errorf("Expected href='https://example.com/download', got '%s' (exists: %v)", href, exists)
	}
}

func TestDocument_FindItemList(t *testing.T) {
	html := `
		<html>
			<body>
				<div class="item-list">
					<div class="post-box-title">
						<a href="https://example.com/post1">Post Title 1</a>
					</div>
					<div class="post-box-title">
						<a href="https://example.com/post2">Post Title 2</a>
					</div>
				</div>
			</body>
		</html>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Test the selector pattern used in cineru and piratelk
	links := doc.Find(".item-list .post-box-title a")
	if len(links.nodes) != 2 {
		t.Errorf("Expected 2 links, got %d", len(links.nodes))
	}

	var titles []string
	var urls []string

	links.Each(func(i int, link *Element) {
		if url, exists := link.Attr("href"); exists {
			urls = append(urls, url)
		}
		title := strings.TrimSpace(link.Text())
		if title != "" {
			titles = append(titles, title)
		}
	})

	expectedTitles := []string{"Post Title 1", "Post Title 2"}
	expectedURLs := []string{"https://example.com/post1", "https://example.com/post2"}

	for i, title := range titles {
		if i < len(expectedTitles) && title != expectedTitles[i] {
			t.Errorf("Expected title '%s', got '%s'", expectedTitles[i], title)
		}
	}

	for i, url := range urls {
		if i < len(expectedURLs) && url != expectedURLs[i] {
			t.Errorf("Expected URL '%s', got '%s'", expectedURLs[i], url)
		}
	}
}

func TestDocument_FindZoomLK(t *testing.T) {
	html := `
		<html>
			<body>
				<div class="td-ss-main-content">
					<div class="item-details">
						<h3 class="entry-title">
							<a href="https://example.com/post1">Post Title 1</a>
						</h3>
					</div>
					<div class="item-details">
						<h3 class="entry-title">
							<a href="https://example.com/post2">Post Title 2</a>
						</h3>
					</div>
				</div>
			</body>
		</html>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Test the selector pattern used in zoomlk
	links := doc.Find(".td-ss-main-content .item-details .entry-title a")
	if len(links.nodes) != 2 {
		t.Errorf("Expected 2 links, got %d", len(links.nodes))
	}

	var titles []string
	links.Each(func(i int, link *Element) {
		title := strings.TrimSpace(link.Text())
		if title != "" {
			titles = append(titles, title)
		}
	})

	expectedTitles := []string{"Post Title 1", "Post Title 2"}
	for i, title := range titles {
		if i < len(expectedTitles) && title != expectedTitles[i] {
			t.Errorf("Expected title '%s', got '%s'", expectedTitles[i], title)
		}
	}
}

func TestDocument_ComplexSelectors(t *testing.T) {
	html := `
		<html>
			<body>
				<div class="container">
					<article class="elementor-post featured">
						<h5 class="elementor-post__title">
							<a href="https://example.com/post1">Featured Post</a>
						</h5>
						<a class="elementor-post__thumbnail__link" href="https://example.com/post1">Thumbnail</a>
					</article>
					<article class="elementor-post">
						<h5 class="elementor-post__title">
							<a href="https://example.com/post2">Regular Post</a>
						</h5>
					</article>
				</div>
			</body>
		</html>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Test multiple classes
	featuredArticles := doc.Find("article.elementor-post.featured")
	if featuredArticles.Len() != 1 {
		t.Errorf("Expected 1 featured article, got %d", featuredArticles.Len())
	}

	// Test that single class still works
	allArticles := doc.Find("article.elementor-post")
	if allArticles.Len() != 2 {
		t.Errorf("Expected 2 articles, got %d", allArticles.Len())
	}

	// Test HasClass method
	featuredEl := featuredArticles.First()
	if !featuredEl.HasClass("featured") {
		t.Error("Featured article should have 'featured' class")
	}
	if !featuredEl.HasClass("elementor-post") {
		t.Error("Featured article should have 'elementor-post' class")
	}
	if featuredEl.HasClass("nonexistent") {
		t.Error("Featured article should not have 'nonexistent' class")
	}
}

func TestDocument_AdvancedSelectors(t *testing.T) {
	html := `
		<html>
			<body>
				<div class="td-ss-main-content">
					<div class="item-details active" data-id="123">
						<h3 class="entry-title">
							<a href="https://example.com/post1" data-track="true">Post 1</a>
						</h3>
					</div>
					<div class="item-details" data-id="456">
						<h3 class="entry-title">
							<a href="https://example.com/post2">Post 2</a>
						</h3>
					</div>
				</div>
				<div id="main-download" data-url="https://download.com/file.zip" class="download-btn">
					<span>Download</span>
				</div>
			</body>
		</html>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Test attribute selector with value
	activeItems := doc.Find("div[data-id=123]")
	if activeItems.Len() != 1 {
		t.Errorf("Expected 1 item with data-id=123, got %d", activeItems.Len())
	}

	// Test attribute selector existence
	itemsWithDataId := doc.Find("div[data-id]")
	if itemsWithDataId.Len() != 2 {
		t.Errorf("Expected 2 items with data-id attribute, got %d", itemsWithDataId.Len())
	}

	// Test ID with classes
	downloadBtn := doc.Find("#main-download.download-btn")
	if downloadBtn.Len() != 1 {
		t.Errorf("Expected 1 download button, got %d", downloadBtn.Len())
	}

	// Test nested attribute selector
	trackedLinks := doc.Find("a[data-track=true]")
	if trackedLinks.Len() != 1 {
		t.Errorf("Expected 1 tracked link, got %d", trackedLinks.Len())
	}

	// Test complex descendant with attribute
	trackedInItems := doc.Find(".item-details a[data-track=true]")
	if trackedInItems.Len() != 1 {
		t.Errorf("Expected 1 tracked link in item-details, got %d", trackedInItems.Len())
	}
}

func TestSelection_UtilityMethods(t *testing.T) {
	html := `
		<div class="container">
			<p class="text">First paragraph</p>
			<p class="text important">Second paragraph</p>
			<p class="text">Third paragraph</p>
		</div>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	paragraphs := doc.Find("p.text")

	// Test Len
	if paragraphs.Len() != 3 {
		t.Errorf("Expected 3 paragraphs, got %d", paragraphs.Len())
	}

	// Test Get
	firstP := paragraphs.Get(0)
	if firstP.Text() != "First paragraph" {
		t.Errorf("Expected 'First paragraph', got '%s'", firstP.Text())
	}

	// Test Last
	lastP := paragraphs.Last()
	if lastP.Text() != "Third paragraph" {
		t.Errorf("Expected 'Third paragraph', got '%s'", lastP.Text())
	}

	// Test Filter
	importantP := paragraphs.Filter(func(el *Element) bool {
		return el.HasClass("important")
	})
	if importantP.Len() != 1 {
		t.Errorf("Expected 1 important paragraph, got %d", importantP.Len())
	}
	if importantP.First().Text() != "Second paragraph" {
		t.Errorf("Expected 'Second paragraph', got '%s'", importantP.First().Text())
	}
}

func TestElement_Methods(t *testing.T) {
	html := `
		<article class="post featured" id="post-123" data-views="1000">
			<h2>Article Title</h2>
			<p>Article content</p>
		</article>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	article := doc.Find("article").First()

	// Test TagName
	if article.TagName() != "article" {
		t.Errorf("Expected tag name 'article', got '%s'", article.TagName())
	}

	// Test Exists
	if !article.Exists() {
		t.Error("Article should exist")
	}

	// Test non-existent element
	nonExistent := doc.Find("nonexistent").First()
	if nonExistent.Exists() {
		t.Error("Non-existent element should not exist")
	}

	// Test multiple attribute access
	if id, exists := article.Attr("id"); !exists || id != "post-123" {
		t.Errorf("Expected id='post-123', got '%s' (exists: %v)", id, exists)
	}

	if views, exists := article.Attr("data-views"); !exists || views != "1000" {
		t.Errorf("Expected data-views='1000', got '%s' (exists: %v)", views, exists)
	}
}

func TestDocument_RealWorldPatterns(t *testing.T) {
	// Test patterns from your actual sources
	html := `
		<html>
			<body>
				<!-- BaiscopeLK pattern -->
				<article class="elementor-post">
					<h5 class="elementor-post__title">
						<a href="https://baiscope.lk/movie1">Movie 1</a>
					</h5>
					<a class="elementor-post__thumbnail__link" href="https://baiscope.lk/movie1">
						<img src="thumb1.jpg" alt="Movie 1">
					</a>
				</article>

				<!-- CineruLK / PirateLK pattern -->
				<div class="item-list">
					<div class="post-box-title">
						<a href="https://cineru.lk/movie2">Movie 2</a>
					</div>
					<div class="post-box-title">
						<a href="https://cineru.lk/movie3">Movie 3</a>
					</div>
				</div>

				<!-- ZoomLK pattern -->
				<div class="td-ss-main-content">
					<div class="item-details">
						<h3 class="entry-title">
							<a href="https://zoom.lk/movie4">Movie 4</a>
						</h3>
					</div>
				</div>

				<!-- Download buttons -->
				<div id="btn-download" data-link="https://download1.com/file.zip">Download 1</div>
				<a data-e-disable-page-transition="true" href="https://download2.com/file.zip">Download 2</a>
				<a class="download-button" href="https://download3.com/file.zip">Download 3</a>
			</body>
		</html>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Test BaiscopeLK selector
	baiscopeLinks := doc.Find("article.elementor-post h5.elementor-post__title a, article.elementor-post a.elementor-post__thumbnail__link")
	if baiscopeLinks.Len() != 2 {
		t.Errorf("Expected 2 baiscope links, got %d", baiscopeLinks.Len())
	}

	// Test CineruLK/PirateLK selector
	itemListLinks := doc.Find(".item-list .post-box-title a")
	if itemListLinks.Len() != 2 {
		t.Errorf("Expected 2 item-list links, got %d", itemListLinks.Len())
	}

	// Test ZoomLK selector
	zoomLinks := doc.Find(".td-ss-main-content .item-details .entry-title a")
	if zoomLinks.Len() != 1 {
		t.Errorf("Expected 1 zoom link, got %d", zoomLinks.Len())
	}

	// Test download button selectors
	downloadBtn1 := doc.Find("#btn-download")
	if link, exists := downloadBtn1.First().Attr("data-link"); !exists || link != "https://download1.com/file.zip" {
		t.Errorf("Expected download1 link, got '%s' (exists: %v)", link, exists)
	}

	downloadBtn2 := doc.Find("a[data-e-disable-page-transition=true]")
	if href, exists := downloadBtn2.First().Attr("href"); !exists || href != "https://download2.com/file.zip" {
		t.Errorf("Expected download2 link, got '%s' (exists: %v)", href, exists)
	}

	downloadBtn3 := doc.Find(".download-button")
	if href, exists := downloadBtn3.First().Attr("href"); !exists || href != "https://download3.com/file.zip" {
		t.Errorf("Expected download3 link, got '%s' (exists: %v)", href, exists)
	}
}

func TestDocument_EdgeCases(t *testing.T) {
	html := `
		<div class="test-class-with-dashes test-second-class">
			<span id="test-id-with-dashes">Test</span>
		</div>
		<div class="">Empty class</div>
		<div>No class</div>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Test class names with dashes
	dashClasses := doc.Find(".test-class-with-dashes")
	if dashClasses.Len() != 1 {
		t.Errorf("Expected 1 element with dashed class, got %d", dashClasses.Len())
	}

	// Test multiple classes with dashes
	multiClass := doc.Find(".test-class-with-dashes.test-second-class")
	if multiClass.Len() != 1 {
		t.Errorf("Expected 1 element with both dashed classes, got %d", multiClass.Len())
	}

	// Test ID with dashes
	dashId := doc.Find("#test-id-with-dashes")
	if dashId.Len() != 1 {
		t.Errorf("Expected 1 element with dashed ID, got %d", dashId.Len())
	}

	// Test empty selector
	empty := doc.Find("")
	if empty.Len() != 0 {
		t.Errorf("Expected 0 elements for empty selector, got %d", empty.Len())
	}
}

func BenchmarkDocument_Find(b *testing.B) {
	html := `
		<html>
			<body>
				<div class="container">
					<article class="elementor-post">
						<h5 class="elementor-post__title">
							<a href="https://example.com/post1">Post Title 1</a>
						</h5>
						<a class="elementor-post__thumbnail__link" href="https://example.com/post1">Thumbnail</a>
					</article>
					<article class="elementor-post">
						<h5 class="elementor-post__title">
							<a href="https://example.com/post2">Post Title 2</a>
						</h5>
					</article>
				</div>
				<div class="item-list">
					<div class="post-box-title">
						<a href="https://example.com/post3">Post Title 3</a>
					</div>
				</div>
				<div id="btn-download" data-link="https://download.com/file.zip">Download</div>
			</body>
		</html>
	`

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		b.Fatalf("Failed to create document: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test common selectors used in your sources
		doc.Find("article.elementor-post")
		doc.Find(".item-list .post-box-title a")
		doc.Find("#btn-download")
		doc.Find("a[data-e-disable-page-transition=true]")
	}
}

func BenchmarkDocument_ComplexSelector(b *testing.B) {
	largeHTML := strings.Repeat(`
		<article class="elementor-post">
			<h5 class="elementor-post__title">
				<a href="https://example.com/post">Post Title</a>
			</h5>
			<a class="elementor-post__thumbnail__link" href="https://example.com/post">Thumbnail</a>
		</article>
	`, 100) // Create 100 articles

	html := "<div class='container'>" + largeHTML + "</div>"

	doc, err := NewDocument(strings.NewReader(html))
	if err != nil {
		b.Fatalf("Failed to create document: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := doc.Find("article.elementor-post h5.elementor-post__title a, article.elementor-post a.elementor-post__thumbnail__link")
		if results.Len() != 200 { // 100 articles * 2 links each
			b.Errorf("Expected 200 results, got %d", results.Len())
		}
	}
}
