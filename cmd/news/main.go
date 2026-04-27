package main

import (
	"log"
	"net/http"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"

	"git.romanzipp.net/romanzipp/news/internal/ai"
	"git.romanzipp.net/romanzipp/news/internal/auth"
	"git.romanzipp.net/romanzipp/news/internal/config"
	"git.romanzipp.net/romanzipp/news/internal/database"
	"git.romanzipp.net/romanzipp/news/internal/digest"
	"git.romanzipp.net/romanzipp/news/internal/handlers"
	"git.romanzipp.net/romanzipp/news/internal/imageproxy"
	"git.romanzipp.net/romanzipp/news/internal/source"
	"git.romanzipp.net/romanzipp/news/internal/templates"
)

func main() {
	godotenv.Load()

	cfg := config.Load()

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, cfg.DBDriver); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// Sessions
	sessions := scs.New()
	sessions.Store = sqlite3store.New(db)
	sessions.Lifetime = 30 * 24 * time.Hour
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.SameSite = http.SameSiteLaxMode

	// Services
	authSvc := auth.NewService(db)
	authMw := auth.NewMiddleware(sessions, authSvc)
	tmpl := templates.New("templates")

	// Source registry
	registry := source.NewRegistry(db)
	registry.Register(&source.RSSProvider{})
	registry.Register(&source.FreshRSSProvider{})

	// AI + Digest
	aiClient := ai.New(cfg)
	gen := digest.NewGenerator(db, cfg, aiClient, registry)
	gen.CleanupStaleJobs()

	// Handlers
	authH := handlers.NewAuthHandler(cfg, authSvc, sessions, tmpl)
	homeH := handlers.NewHomeHandler(db, sessions, tmpl, gen)
	digestH := handlers.NewDigestHandler(sessions, tmpl, gen)
	feedsH := handlers.NewFeedsHandler(db, sessions, tmpl, registry)
	interestsH := handlers.NewInterestsHandler(db, sessions, tmpl)
	sectionsH := handlers.NewSectionsHandler(db, sessions, tmpl)
	votesH := handlers.NewVotesHandler(db, tmpl)
	readH := handlers.NewReadHandler(db)
	copyH := handlers.NewCopyHandler(db, tmpl)

	// Routes
	mux := http.NewServeMux()

	// Static
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Auth (public)
	mux.HandleFunc("GET /login", authH.LoginPage)
	mux.HandleFunc("POST /login", authH.LoginSubmit)
	mux.HandleFunc("GET /register", authH.RegisterPage)
	mux.HandleFunc("POST /register", authH.RegisterSubmit)
	mux.HandleFunc("POST /logout", authH.Logout)

	// Image proxy (public)
	mux.HandleFunc("GET /proxy/image", imageproxy.Handler)

	// Authenticated routes
	mux.Handle("GET /{$}", authMw.RequireAuth(http.HandlerFunc(homeH.Home)))
	mux.Handle("GET /digest/{date}", authMw.RequireAuth(http.HandlerFunc(homeH.DigestByDate)))
	mux.Handle("GET /digest/{date}/{id}", authMw.RequireAuth(http.HandlerFunc(homeH.DigestByID)))
	mux.Handle("POST /digest/generate", authMw.RequireAuth(http.HandlerFunc(digestH.Generate)))
	mux.Handle("GET /digest/generating/{jobID}", authMw.RequireAuth(http.HandlerFunc(digestH.GeneratingPage)))
	mux.Handle("GET /digest/generating/{jobID}/status", authMw.RequireAuth(http.HandlerFunc(digestH.GeneratingStatus)))

	mux.Handle("GET /feeds", authMw.RequireAuth(http.HandlerFunc(feedsH.FeedsPage)))
	mux.Handle("POST /feeds", authMw.RequireAuth(http.HandlerFunc(feedsH.FeedAdd)))
	mux.Handle("POST /feeds/{id}/delete", authMw.RequireAuth(http.HandlerFunc(feedsH.FeedDelete)))
	mux.Handle("POST /feeds/{id}/toggle", authMw.RequireAuth(http.HandlerFunc(feedsH.FeedToggle)))
	mux.Handle("POST /feeds/fetch", authMw.RequireAuth(http.HandlerFunc(feedsH.FetchNow)))

	mux.Handle("GET /freshrss", authMw.RequireAuth(http.HandlerFunc(feedsH.FreshRSSPage)))
	mux.Handle("POST /freshrss", authMw.RequireAuth(http.HandlerFunc(feedsH.FreshRSSSave)))
	mux.Handle("POST /freshrss/test", authMw.RequireAuth(http.HandlerFunc(feedsH.FreshRSSTest)))

	mux.Handle("GET /wizard", authMw.RequireAuth(http.HandlerFunc(interestsH.WizardPage)))
	mux.Handle("POST /wizard", authMw.RequireAuth(http.HandlerFunc(interestsH.WizardSave)))
	mux.Handle("GET /interests", authMw.RequireAuth(http.HandlerFunc(interestsH.InterestsPage)))
	mux.Handle("POST /interests", authMw.RequireAuth(http.HandlerFunc(interestsH.InterestsSave)))

	mux.Handle("GET /sections", authMw.RequireAuth(http.HandlerFunc(sectionsH.SectionsPage)))
	mux.Handle("POST /sections", authMw.RequireAuth(http.HandlerFunc(sectionsH.SectionAdd)))
	mux.Handle("POST /sections/{id}/update", authMw.RequireAuth(http.HandlerFunc(sectionsH.SectionUpdate)))
	mux.Handle("POST /sections/{id}/delete", authMw.RequireAuth(http.HandlerFunc(sectionsH.SectionDelete)))

	mux.Handle("POST /votes", authMw.RequireAuth(http.HandlerFunc(votesH.Vote)))

	mux.Handle("GET /read/{id}", authMw.RequireAuth(http.HandlerFunc(readH.MarkAndRedirect)))
	mux.Handle("GET /copy/{id}", authMw.RequireAuth(http.HandlerFunc(copyH.CopyMarkdown)))

	// Cron scheduler
	c := cron.New()
	c.AddFunc(cfg.FetchCron, func() {
		log.Println("cron: fetching feeds")
		registry.FetchAllUsers(nil)
	})
	c.AddFunc(cfg.DigestCron, func() {
		log.Println("cron: generating digests")
		gen.GenerateAllUsers(nil)
	})
	c.Start()
	defer c.Stop()

	// Wrap with session middleware + user loading
	handler := sessions.LoadAndSave(authMw.LoadUser(mux))

	log.Printf("listening on %s", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, handler))
}
