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
	"github.com/russross/blackfriday"
)

type (
	config struct {
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
	GOW_TITLE  = "Go Wiki"
	GOW_HEADER = `<!DOCTYPE html><html><head><title>` + GOW_TITLE + `</title></head>
	<style>
	* { color:#3F3F3F; font-family:"Lucida Sans Unicode", "Lucida Grande", sans-serif; }
	a { color:#47D2E9; text-decoration:none; font-weight:bold; }
	nav ul { list-style-type:none; padding:0; }
	nav ul li { display:inline; margin:0 10px; }
	ul.tools {display: none;list-style: none;box-shadow: 0px 0px 4px rgba(0,0,0,.5);border: solid 1px #000;position: absolute;background: #fff;padding:0;}
	ul.tools li {display: inline-block;height: 20px;border: solid 1px #000;margin: 5px;padding: 5px 10px;cursor: pointer;}
	#go-back-link a {background-color:#EEEEEE;padding:5px 12px;font-weight:500;font: 13.3333px Arial;color:#3F3F3F;border:1px solid #dedede;}
	</style>
	<body>
	<header style="border-bottom:1px solid #EEEEEE;height:30px;padding-top:10px;">
		<div style="margin: 0 10px;">
			<a href="/">` + GOW_TITLE + `</a>
		</div>
	</header>
	<nav>
		<ul>
			<li>
				<a href="/create">Create new article</a>
			</li>
			 <li>
				 <a href="/list">List articles</a>
			 </li>
		</ul>
	</nav>
	<div style="padding: 10px 0;"><div style="margin: 0 10px;">`
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
										'left': pageX + 5,
										'top' : pageY - 55
								}).fadeIn(200);
								var urlC = encodeURIComponent(txt);
								$('#tools-create-link').attr('href', '/create?t='+urlC);
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
	flag.StringVar(&Cfg.Port, "port", "9090", "the port you want to run GOW on")
	flag.StringVar(&Cfg.Bucket, "bucket", "./gow.bucket", "the folder in which data should be stored")
	flag.StringVar(&Cfg.Key, "key", "d51b2bf666420e87ab91d08ef07f2e08", "the secret key you want to use for encryption")
	flag.Parse()
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
	r.HandleFunc("/list", listAllHandler).Methods("GET")
	r.HandleFunc("/create", createHandler).Methods("GET")
	r.HandleFunc("/create", createPostHandler).Methods("POST")
	r.HandleFunc("/article/{link}", viewHandler).Methods("GET")
	r.HandleFunc("/delete/{link}", deleteHandler).Methods("GET")

	http.Handle("/", r)
	err := http.ListenAndServe(fmt.Sprintf(":%s", Cfg.Port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t := template.New("indexTpl")
	MyHtml := GOW_HEADER + "hej" + GOW_FOOTER
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
	unsafe := blackfriday.MarkdownCommon(plaintextCopy)
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	stringHtml := string(html)
	for _, v := range headers {
		if v.Link != link {
			stringHtml = strings.Replace(stringHtml, v.Title, fmt.Sprintf("<a href='/article/%s'>%s</a>", v.Link, v.Title), -1)
		}
	}
	MyHtml := GOW_HEADER + `
			<h1>
				` + article.Title + `
			</h1>
			<small><em>Added/Updated ` + article.Created + `</em></small>
			<p>
				` + stringHtml + `
			</p>
			<a href="/delete/` + article.Link + `" style="float:right;position:absolute;bottom:10px;left:20px;">Delete article</a>
		` + GOW_FOOTER
	t, _ = t.Parse(MyHtml)
	t.Execute(w, nil)
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

func listAllHandler(w http.ResponseWriter, r *http.Request) {
	var (
		tplArticles []string
		articles    []article
	)
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
	sort.Sort(articlesDescending(articles))
	for _, article := range articles {
		tplArticles = append(tplArticles, fmt.Sprintf("<li><a href='/article/%s'>%s</a></li>", article.Link, article.Title))
	}
	joinedArticlesList := strings.Join(tplArticles, "")
	t := template.New("listAllTpl")
	MyHtml := GOW_HEADER + `
	 	<ul>
	 		` + joinedArticlesList + `
	 	</ul>
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
			MyHtml = MyHtml + `<div id="go-back-link">
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
