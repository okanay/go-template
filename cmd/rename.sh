#!/bin/bash

# chmod +x ./cmd/rename.sh
# ./cmd/rename.sh github.com/okanay/go-template


# Hata durumunda dur
set -e

# Mevcut modül adı (go.mod dosyasından otomatik bulur)
CURRENT_MODULE=$(grep "^module" go.mod | awk '{print $2}')

if [ -z "$1" ]; then
    echo "❌ Hata: Yeni modül adını belirtmediniz."
    echo "Kullanım: ./init-project.sh github.com/kullanici/yeni-proje"
    exit 1
fi

NEW_MODULE=$1

echo "🚀 Proje ismi değiştiriliyor..."
echo "Eski Modül: $CURRENT_MODULE"
echo "Yeni Modül: $NEW_MODULE"
echo "--------------------------------------------------"

# 1. go.mod dosyasını güncelle
echo "📄 go.mod güncelleniyor..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i "" "s|$CURRENT_MODULE|$NEW_MODULE|g" go.mod
else
    sed -i "s|$CURRENT_MODULE|$NEW_MODULE|g" go.mod
fi

# 2. Tüm .go dosyalarındaki importları güncelle (Döngü ile - Daha Güvenli)
echo "🔄 Import yolları güncelleniyor..."

# Gereksiz klasörleri hariç tutarak .go dosyalarını buluyoruz
find . -type f -name "*.go" \
    -not -path "*/vendor/*" \
    -not -path "*/.git/*" \
    -not -path "*/node_modules/*" \
    -not -path "*/bin/*" \
    -not -path "*/.idea/*" \
    -not -path "*/.vscode/*" | while read -r file; do

    # Her dosya için sed komutunu çalıştır
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i "" "s|$CURRENT_MODULE|$NEW_MODULE|g" "$file"
    else
        sed -i "s|$CURRENT_MODULE|$NEW_MODULE|g" "$file"
    fi
done

# 3. go mod tidy çalıştır
echo "🧹 Bağımlılıklar temizleniyor (go mod tidy)..."
go mod tidy

echo "--------------------------------------------------"
echo "✅ Başarılı! Proje '$NEW_MODULE' olarak güncellendi."
