package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dvwallin/gow/Lib"
	"github.com/gorilla/mux"
	"github.com/microcosm-cc/bluemonday"
	scribble "github.com/nanobox-io/golang-scribble"
	"github.com/renstrom/fuzzysearch/fuzzy"
	"gopkg.in/russross/blackfriday.v2"
)

type (
	config struct {
		Host   string
		Port   string
		Bucket string
		Key    string
	}
	article struct {
		Link    string
		Title   string
		Content []byte
		Created string
		PTL     int
	}
	articlesDescending []article
)

func (v articlesDescending) Len() int           { return len(v) }
func (v articlesDescending) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (a articlesDescending) Less(i, j int) bool { return a[i].Title > a[j].Title }

const (
	SCHEME     = "http"
	GOW_TITLE  = "GOW"
	GOW_HEADER = `<!DOCTYPE html><html><head><title>` + GOW_TITLE + `</title></head>
	<style>
	@import url('https://fonts.googleapis.com/css?family=Libre+Baskerville:700|Montserrat:300,500');body,html{height:100%;width:100%;margin:0;padding:0;left:0;top:0;font-size:100%}.center,.container{margin-left:auto;margin-right:auto}*{font-family:'Montserrat', sans-serif;color:#333447;line-height:1.5}h1,h2,h3,h4,h5,h6{font-family:'Libre Baskerville', serif;}h1{font-size:3.5rem}h2{font-size:2.5rem}h3{font-size:1.375rem}h4{font-size:1.125rem}h5{font-size:1rem}h6{font-size:.875rem}p{font-size:1.125rem;font-weight:200;line-height:1.8}.font-light{font-weight:300}.font-regular{font-weight:400}.font-heavy{font-weight:700}.left{text-align:left}.right{text-align:right}.center{text-align:center}.justify{text-align:justify}.container{width:90%}.row{position:relative;width:100%}.row [class^=col]{float:left;margin:.5rem 2%;min-height:.125rem}.col-1,.col-10,.col-11,.col-12,.col-2,.col-3,.col-4,.col-5,.col-6,.col-7,.col-8,.col-9{width:96%}.col-1-sm{width:4.33%}.col-2-sm{width:12.66%}.col-3-sm{width:21%}.col-4-sm{width:29.33%}.col-5-sm{width:37.66%}.col-6-sm{width:46%}.col-7-sm{width:54.33%}.col-8-sm{width:62.66%}.col-9-sm{width:71%}.col-10-sm{width:79.33%}.col-11-sm{width:87.66%}.col-12-sm{width:96%}.row::after{content:"";display:table;clear:both}.hidden-sm{display:none}@media only screen and (min-width:33.75em){.container{width:80%}}@media only screen and (min-width:45em){.col-1{width:4.33%}.col-2{width:12.66%}.col-3{width:21%}.col-4{width:29.33%}.col-5{width:37.66%}.col-6{width:46%}.col-7{width:54.33%}.col-8{width:62.66%}.col-9{width:71%}.col-10{width:79.33%}.col-11{width:87.66%}.col-12{width:96%}.hidden-sm{display:block}}@media only screen and (min-width:60em){.container{width:75%;max-width:60rem}}
	.tools { display: none; position: absolute; list-style: none; padding: 0.8rem; background: #f4f4f4; border: 1px solid #ccc; }
	.tools a { text-decoration: none; }
	nav > ul { padding: 0; }
	nav > ul > li { display: inline; }
	.search_form > p > input[type="text"] { padding: 0.2rem; font-size: 1rem; border: 1px solid #ccc; min-width: 80%; }
	.search_form > p > button {padding: 0.2rem; font-size: 1rem; background: #fff; border: 1px solid #ccc;}
	.go-back-link {display:inline-block; padding: 0.2rem; font-size: 1rem;background: #f4f4f4; border: 1px solid #ccc;}
	.go-back-link a {text-decoration: none;}
	.row > div.no-margin { margin: 0; }
	body { padding-top: 2rem; }
	</style>
	<body>
	<div class="container">
		<div class="row">
			<div class="col-12 no-margin">
				<div class="col-3">
					<nav>
						<ul>
							<li>
								<div class="go-back-link">
									<a href="/">` + GOW_TITLE + `</a>
								</div>
							</li>
							<li>
								<div class="go-back-link">
									<a href="/create" title="Create new article">New</a>
								</div>
							</li>
						</ul>
					</nav>
				</div>
				<div class="col-9">
					<form name="search" method="POST" action="/search" class="search_form">
						<p>
							<input type="text" name="search_term" placeholder="Enter something to search for..." pattern="[a-zA-Z0-9åäöÅÄÖ,_.-/\:+?! ]{2,250}"/>
							<button type="submit" name="submit">Search</button>
						</p>
					</form>
				</div>
			</div>
		</div>
		<div class="row">
		</div>
		<div class="row">
			<div class="col-12">`
	GOW_FOOTER = `</div></div>
	
	
		<ul class="tools">
			<li><a href="/create" id="tools-create-link">Create</a></li>
		</ul>
		<script type='text/javascript'>
			` + Lib.JQ + `
		</script>
		<script type='text/javascript'>
		
		function getSelectionText() {
				var text = "";
				var activeEl = document.activeElement;
				var activeElTagName = activeEl ? activeEl.tagName.toLowerCase() : null;
				if (
					(activeElTagName == "textarea" || activeElTagName == "input") &&
					/^(?:text|search|password|tel|url)$/i.test(activeEl.type) &&
					(typeof activeEl.selectionStart == "number")
				) {
					text = activeEl.value.slice(activeEl.selectionStart, activeEl.selectionEnd);
				} else if (window.getSelection) {
						text = window.getSelection().toString();
				}
				return text;
		}
		
		if (!window.x) {
				x = {};
		}
		
		x.Selector = {};
		x.Selector.getSelected = function() {
				var t = '';
				if (window.getSelection) {
						t = window.getSelection();
				} else if (document.getSelection) {
						t = document.getSelection();
				} else if (document.selection) {
						t = document.selection.createRange().text;
				}
				return t;
		}
		
		var pageX;
		var pageY;
		
		$(document).ready(function() {
				$(document).bind("mouseup", function() {
						var selectedText = x.Selector.getSelected();
						var txt = getSelectionText();
						if(selectedText != ''){
								$('ul.tools').css({
										'left': pageX,
										'top' : pageY + 5
								}).fadeIn(200);
								var urlC = encodeURIComponent(txt);
								$('#tools-create-link').attr('href', '/create?t='+urlC);
								$('#tools-create-link').text('Click here to create new article called \"'+txt+'\"');
						} else {
								$('ul.tools').fadeOut(200);
						}
				});
				$(document).on("mousedown", function(e){
						pageX = e.pageX;
						pageY = e.pageY;
				});
		});
	</script></body></html>`
)

var (
	Cfg      config
	DB       *scribble.Driver
	commonIV []byte
	c        cipher.Block
	err      error
)

func init() {
	flag.StringVar(&Cfg.Host, "host", "0.0.0.0", "the host on which to host the web interface")
	flag.StringVar(&Cfg.Port, "port", "9090", "the port you want to run GOW on")
	flag.StringVar(&Cfg.Bucket, "bucket", "./gow.bucket", "the folder in which data should be stored")
	flag.StringVar(&Cfg.Key, "key", "d51b2bf666420e87ab91d08ef07f2e08", "the secret key you want to use for encryption")
	flag.Parse()
	keyLength := len(Cfg.Key)
	if keyLength != 32 {
		fmt.Println("The security key must be 32 characters long")
		if keyLength > 32 {
			fmt.Printf("You gave %d too many characters\n", (keyLength - 32))
		}
		if keyLength < 32 {
			fmt.Printf("You gave %d too few characters\n", (32 - keyLength))
		}
		os.Exit(1)
	}
	DB, _ = scribble.New(Cfg.Bucket, nil)
	commonIV = []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	c, err = aes.NewCipher([]byte(Cfg.Key))
	if err != nil {
		fmt.Printf("Error: NewCipher(%d bytes) = %s", len(Cfg.Key), err)
		os.Exit(-1)
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/create", createHandler).Methods("GET")
	r.HandleFunc("/create", createPostHandler).Methods("POST")
	r.HandleFunc("/article/{link}", viewHandler).Methods("GET")
	r.HandleFunc("/delete/{link}", deleteHandler).Methods("GET")
	r.HandleFunc("/edit/{link}", editHandler).Methods("GET")
	r.HandleFunc("/edit/{link}", editPostHandler).Methods("POST")
	r.HandleFunc("/search", searchHandler).Methods("GET")
	r.HandleFunc("/search", searchPostHandler).Methods("POST")

	http.Handle("/", r)
	fmt.Printf("Running %s on %s:%s", GOW_TITLE, Cfg.Host, Cfg.Port)
	err := http.ListenAndServe(fmt.Sprintf("%s:%s", Cfg.Host, Cfg.Port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var (
		tplArticles []string
		articles    []article
		headers = getAllHeaders()
	)
	records, _ := DB.ReadAll("articles")
	for _, v := range records {
		article := article{}
		if err := json.Unmarshal([]byte(v), &article); err != nil {
			fmt.Println("Error", err)
		}
		articles = append(articles, article)
	}
	sort.Sort(articlesDescending(articles))
	for _, article := range articles {
		tplArticles = append(tplArticles, fmt.Sprintf("<a href='/article/%s'>%s</a>", article.Link, article.Title))
	}
	joinedArticlesList := strings.Join(tplArticles, " - ")
	t := template.New("indexTpl")
	MyHtml := GOW_HEADER
	content := `
		<h1>Welcome to ` + GOW_TITLE + `</h1>
		<p>` + GOW_TITLE + ` is a small standalone wiki-system mean for lokal portable documentation.
		When starting the application you supply a key which is your personal encryption key. All article content (not the title) is then encrypted using this key and thus only you (or the ones you hand the key to) may access the data.</p>
		<hr />
		<h2>Articles</h2>
		<p>`+ joinedArticlesList +`</p>
	`
	for _, v := range headers {
		content = strings.Replace(content, v.Title, fmt.Sprintf("<a href='/article/%s'>%s</a>", v.Link, v.Title), -1)
	}
	MyHtml = MyHtml + content
	MyHtml = MyHtml + GOW_FOOTER
	t, _ = t.Parse(MyHtml)
	t.Execute(w, nil)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	link := vars["link"]
	confirmGet := r.URL.Query().Get("confirmed")
	if confirmGet == "yes" {
		if err := DB.Delete("articles", link); err != nil {
			fmt.Println("Error", err)
		}
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	} else {
		t := template.New("deleteTpl")
		MyHtml := GOW_HEADER + `
		<h3>Are you sure?</h3>
		<p>
			Deleting an article is permanent and cannot be undone.
		</p>
		<p>
			<a href="/article/` + link + `">Abort</a> <a style="margin-left: 50px;" href="/delete/` + link + `?confirmed=yes">Proceed</a>
		</p>
	` + GOW_FOOTER
		t, _ = t.Parse(MyHtml)
		t.Execute(w, nil)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	link := vars["link"]
	article := article{}
	headers := getAllHeaders()
	if err := DB.Read("articles", link, &article); err != nil {
		fmt.Println("Error", err)
	}
	t := template.New("viewTpl")
	cfbdec := cipher.NewCFBDecrypter(c, commonIV)
	plaintextCopy := make([]byte, article.PTL)
	cfbdec.XORKeyStream(plaintextCopy, article.Content)
	unsafe := blackfriday.Run(plaintextCopy)
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	stringHtml := string(html)
	for _, v := range headers {
		if v.Link != link {
			stringHtml = strings.Replace(stringHtml, v.Title, fmt.Sprintf("<a href='/article/%s'>%s</a>", v.Link, v.Title), -1)
		}
	}
	MyHtml := GOW_HEADER + `
			<div class="go-back-link">
				<a href="/edit/` + article.Link + `">Edit</a>
			</div>
			<h1>
				` + article.Title + `
			</h1>
			<small><em>Added/Updated ` + article.Created + `</em></small>
			<p>
				` + stringHtml + `
			</p>
		` + GOW_FOOTER
	t, _ = t.Parse(MyHtml)
	t.Execute(w, nil)
}

func createPostHandler(w http.ResponseWriter, r *http.Request) {
	title := base64.StdEncoding.EncodeToString([]byte(r.FormValue("title")))
	cfb := cipher.NewCFBEncrypter(c, commonIV)
	ciphertext := make([]byte, len(r.FormValue("content")))
	cfb.XORKeyStream(ciphertext, []byte(r.FormValue("content")))
	article := article{
		Link:    title,
		Title:   r.FormValue("title"),
		Content: ciphertext,
		Created: time.Now().String(),
		PTL:     len(r.FormValue("content")),
	}
	if err := DB.Write("articles", title, article); err != nil {
		fmt.Println("Error", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/article/%s", article.Link), http.StatusMovedPermanently)
}

func getCurrURL(r *http.Request) (cURL string) {
	h := r.Host
	t := r.URL.String()
	cURL = fmt.Sprintf("%s://%s%s", SCHEME, h, t)
	return
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	var presetTitle string
	sentTitle := r.URL.Query().Get("t")
	if sentTitle != "" && len(sentTitle) > 2 {
		presetTitle = sentTitle
	}
	t := template.New("createTpl")
	MyHtml := GOW_HEADER

	if presetTitle != "" && len(presetTitle) > 2 {
		referer := r.Referer()
		currURL := getCurrURL(r)
		if referer != currURL {
			MyHtml = MyHtml + `<div class="go-back-link">
			<a href="` + referer + `">Go back</a>
		</div>`
		}
	}

	MyHtml = MyHtml + `
	  <form name="create" method="POST" action="">
  		<p>
			<input type="text" name="title" style="width:90%;padding:5px;" value="` + presetTitle + `" placeholder="Title..." pattern="[a-zA-Z0-9åäöÅÄÖ,_.-/\:+?! ]{2,250}" />
		</p>
		<p>
			<textarea name="content" placeholder="Article content..." style="width:90%; height:350px; padding:5px;"></textarea>
		</p>
		<p>
			<button type="submit" name="submit" style="background:#53DF83;padding:6px 12px;border:0;">Save</button>
		</p>
	  </form>
	  ` + GOW_FOOTER
	t, _ = t.Parse(MyHtml)
	t.Execute(w, nil)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	link := vars["link"]
	article := article{}
	if err := DB.Read("articles", link, &article); err != nil {
		fmt.Println("Error", err)
	}
	t := template.New("editTpl")
	cfbdec := cipher.NewCFBDecrypter(c, commonIV)
	plaintextCopy := make([]byte, article.PTL)
	cfbdec.XORKeyStream(plaintextCopy, article.Content)
	stringHtml := string(plaintextCopy)

	MyHtml := GOW_HEADER

	MyHtml = MyHtml + `<div>
			<a href="/article/` + link + `">Abort</a> - 
			<a href="/delete/` + article.Link + `">Delete</a>
		</div>
	  <form name="edit" method="POST" action="">
  		<p>
			<input type="text" name="title" style="width:90%;padding:5px;" value="` + article.Title + `" placeholder="Title..." pattern="[a-zA-Z0-9åäöÅÄÖ,_.-/\:+?! ]{2,250}" />
		</p>
		<p>
			<textarea name="content" placeholder="Article content..." style="width:90%; height:350px; padding:5px;">` + stringHtml + `</textarea>
		</p>
		<p>
			<button type="submit" name="submit" style="background:#53DF83;padding:6px 12px;border:0;">Save</button>
		</p>
	  </form>
	  ` + GOW_FOOTER
	t, _ = t.Parse(MyHtml)
	t.Execute(w, nil)
}

func editPostHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	link := vars["link"]
	title := base64.StdEncoding.EncodeToString([]byte(r.FormValue("title")))

	if link != title {
		if err := DB.Delete("articles", link); err != nil {
			fmt.Println("Error", err)
		}
		link = title
	}

	cfb := cipher.NewCFBEncrypter(c, commonIV)
	ciphertext := make([]byte, len(r.FormValue("content")))
	cfb.XORKeyStream(ciphertext, []byte(r.FormValue("content")))
	article := article{
		Link:    link,
		Title:   r.FormValue("title"),
		Content: ciphertext,
		Created: time.Now().String(),
		PTL:     len(r.FormValue("content")),
	}
	if err := DB.Write("articles", title, article); err != nil {
		fmt.Println("Error", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/article/%s", article.Link), http.StatusMovedPermanently)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	t := template.New("searchTpl")
	MyHtml := GOW_HEADER
	MyHtml = MyHtml + `
	  <form name="search" method="POST" action="">
  		<p>
			<input type="text" name="search_term" style="width:80%;padding:5px;" placeholder="What are you looking for?" pattern="[a-zA-Z0-9åäöÅÄÖ,_.-/\:+?! ]{2,250}" />
			<button type="submit" name="submit" style="background:#53DF83;padding:6px 12px;border:0;">Search</button>
		</p>
	  </form>
	  ` + GOW_FOOTER
	t, _ = t.Parse(MyHtml)
	t.Execute(w, nil)
}

func searchPostHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.FormValue("search_term")
	headers := getAllHeaders()
	var headerTitles []string
	for _, v := range headers {
		headerTitles = append(headerTitles, v.Title)
	}
	matches := fuzzy.RankFindFold(searchTerm, headerTitles)
	t := template.New("searchTpl")
	MyHtml := GOW_HEADER
	var articleList []string
	counter := 1
	for _, v := range matches {
		article := getArticleByHeader(v.Target)
		articleList = append(articleList, `<div class="article-item">
			<h5>#`+fmt.Sprintf("%d", counter)+`. <a href="/article/`+article.Link+`">`+article.Title+`</a></h5>
		</div>`)
		counter++
	}
	MyHtml = MyHtml + strings.Join(articleList, "")
	MyHtml = MyHtml + GOW_FOOTER
	t, _ = t.Parse(MyHtml)
	t.Execute(w, nil)
}

func getArticleByHeader(h string) (article article) {
	headers := getAllHeaders()
	for _, v := range headers {
		if v.Title == h {
			article = v
		}
	}
	return
}

func getAllHeaders() (articles []article) {
	records, err := DB.ReadAll("articles")
	if err != nil {
		fmt.Println("Error", err)
	}
	for _, v := range records {
		article := article{}
		if err := json.Unmarshal([]byte(v), &article); err != nil {
			fmt.Println("Error", err)
		}
		articles = append(articles, article)
	}
	return
}
