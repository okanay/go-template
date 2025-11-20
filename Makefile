## ----------------------------------------------------------------------
## Proje Değişkenleri
## ----------------------------------------------------------------------

BINARY_NAME=main
CMD_PATH=main.go

# .PHONY ile make hedeflerinin dosya ismi olmadığını belirtiyoruz
.PHONY: run build dev clean help

# Varsayılan hedef (sadece 'make' yazınca çalışır)
all: help

## ----------------------------------------------------------------------
## Proje Komutları
## ----------------------------------------------------------------------

# Uygulamayı hot-reload modu ile çalıştır (Air kullanarak)
# Gereksinim: 'air' yüklü olmalıdır. (go install github.com/air-verse/air@latest)
dev:
	@echo "🔄 Geliştirme modu (Hot-Reload) başlatılıyor..."
	air

# Uygulamayı derle (build) ve bin/ klasörüne çıktı al
build:
	@echo "🔨 Uygulama derleniyor..."
	@mkdir -p bin
	go build -o bin/$(BINARY_NAME) $(CMD_PATH)
	@echo "✅ Derleme tamamlandı: bin/$(BINARY_NAME)"

# Uygulamayı normal şekilde çalıştır (go run)
run:
	@echo "🚀 Uygulama başlatılıyor..."
	go run $(CMD_PATH)

# Derlenmiş dosyaları ve geçici dosyaları temizle
clean:
	@echo "🧹 Temizlik yapılıyor..."
	@rm -f bin/$(BINARY_NAME)
	@rm -rf tmp
	@echo "✅ Temizlendi."

# Yardım menüsü
help:
	@echo "Kullanılabilir Komutlar:"
	@echo "  make dev    - Uygulamayı Air ile hot-reload modunda başlatır (Önerilen)"
	@echo "  make run    - Uygulamayı normal modda başlatır"
	@echo "  make build  - Uygulamayı derler (bin/ klasörüne)"
	@echo "  make clean  - Derlenmiş dosyaları temizler"
