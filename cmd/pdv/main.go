// Command pdv is the Soparia PDV backend: a single self-contained binary
// (frontend embedded, SQLite database next to the .exe) that replaces the
// Node/Express server from src/server.js, so running the PDV on the
// shop's computer no longer needs Node installed at all.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	webassets "github.com/felpsalvs/pdv-starter"
	"github.com/felpsalvs/pdv-starter/internal/backup"
	"github.com/felpsalvs/pdv-starter/internal/config"
	"github.com/felpsalvs/pdv-starter/internal/db"
	"github.com/felpsalvs/pdv-starter/internal/httpapi"
	"github.com/felpsalvs/pdv-starter/internal/printer"
	"github.com/felpsalvs/pdv-starter/internal/service"
	"github.com/felpsalvs/pdv-starter/internal/store"
	"github.com/felpsalvs/pdv-starter/internal/update"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z" by
// .goreleaser.yaml. A "go build"/"go run" without that flag stays "dev",
// which disables auto-update (see internal/update.CheckLatest).
var version = "dev"

func main() {
	noUpdate := flag.Bool("no-update", false, "não verificar/aplicar atualizações ao iniciar")
	noBrowser := flag.Bool("no-browser", false, "não abrir o navegador automaticamente")
	flag.Parse()

	if os.Getenv("PDV_NO_UPDATE") == "1" {
		*noUpdate = true
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("não foi possível resolver os caminhos de configuração: %v", err)
	}

	if !*noUpdate {
		maybeUpdateAndRestart(cfg)
	}
	if exePath, err := os.Executable(); err == nil {
		update.CleanupPrevious(exePath)
	}

	sqlDB, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("não foi possível abrir o banco de dados: %v", err)
	}
	defer sqlDB.Close()

	run(cfg, sqlDB, *noBrowser)
}

func maybeUpdateAndRestart(cfg config.Config) {
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("não foi possível localizar o executável atual, pulando verificação de atualização: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	release, hasUpdate, err := update.CheckLatest(ctx, version)
	if err != nil {
		log.Printf("não foi possível verificar atualizações (seguindo offline): %v", err)
		return
	}
	if !hasUpdate {
		return
	}

	log.Printf("Nova versão %s disponível (atual: %s). Baixando e aplicando...", release.TagName, version)

	// Snapshot the database before touching the binary, so an update that
	// goes wrong never costs the day's data.
	if sqlDB, err := db.Open(cfg.DBPath); err == nil {
		if err := backup.New(sqlDB, cfg.BackupDir).Create(); err != nil {
			log.Printf("backup pré-atualização falhou, mantendo a versão atual: %v", err)
			sqlDB.Close()
			return
		}
		sqlDB.Close()
	} else {
		log.Printf("não foi possível abrir o banco para o backup pré-atualização, mantendo a versão atual: %v", err)
		return
	}

	if err := update.DownloadAndApply(ctx, release, exePath); err != nil {
		log.Printf("falha ao aplicar a atualização, mantendo a versão atual: %v", err)
		return
	}

	log.Printf("Atualização aplicada. Reiniciando...")
	if err := update.Restart(exePath, os.Args[1:]); err != nil {
		log.Printf("atualização baixada, mas não foi possível reiniciar automaticamente: %v. Feche e abra o programa de novo.", err)
		os.Exit(0)
	}
	os.Exit(0)
}

func run(cfg config.Config, sqlDB *sql.DB, noBrowser bool) {
	st := store.New(sqlDB)
	prn := printer.New(cfg.PrinterConfig)
	ordersService := service.NewOrdersService(st, prn)
	cashRegisterService := service.NewCashRegisterService(st)
	handlers := httpapi.NewHandlers(st, ordersService, cashRegisterService)

	distFS, err := fs.Sub(webassets.Dist, "dist")
	if err != nil {
		log.Fatalf("não foi possível carregar os arquivos do frontend embutidos: %v", err)
	}
	static := http.FileServerFS(distFS)

	router := httpapi.NewRouter(handlers, static)
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	backupper := backup.New(sqlDB, cfg.BackupDir)
	backupper.Schedule(ctx)

	go func() {
		url := fmt.Sprintf("http://localhost:%s", cfg.Port)
		log.Printf("Soparia PDV rodando em %s", url)
		if !noBrowser {
			openBrowser(url)
		}
	}()

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("erro no servidor HTTP: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("erro ao encerrar o servidor: %v", err)
	}

	if err := backupper.Create(); err != nil {
		log.Println(err)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("não foi possível abrir o navegador automaticamente: %v", err)
	}
}
