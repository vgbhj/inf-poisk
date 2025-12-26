#!/bin/bash

HLTV_DIR="corpus/hltv"
CYBERSPORT_DIR="corpus/cybersport"

# Счетчик удаленных файлов
deleted=0
empty_deleted=0

echo "=== Cloudflare cleanup ==="

# Удаление из HLTV
if [ -d "$HLTV_DIR" ]; then
    echo "Scanning $HLTV_DIR for Cloudflare blocks..."
    while IFS= read -r file; do
        if grep -q "Verify you are human by completing the action below" "$file"; then
            rm "$file"
            echo "Deleted (Cloudflare): $file"
            ((deleted++))
        fi
    done < <(find "$HLTV_DIR" -type f -name "*.txt")
fi

# Удаление из Cybersport
if [ -d "$CYBERSPORT_DIR" ]; then
    echo "Scanning $CYBERSPORT_DIR for Cloudflare blocks..."
    while IFS= read -r file; do
        if grep -q "Verify you are human by completing the action below" "$file"; then
            rm "$file"
            echo "Deleted (Cloudflare): $file"
            ((deleted++))
        fi
    done < <(find "$CYBERSPORT_DIR" -type f -name "*.txt")
fi

echo "Total Cloudflare deleted: $deleted files"

echo ""
echo "=== Empty files cleanup ==="

# Удаление пустых файлов из HLTV
if [ -d "$HLTV_DIR" ]; then
    echo "Scanning $HLTV_DIR for empty files..."
    while IFS= read -r file; do
        if awk '
            BEGIN { cnt = 0 }
            {
                if ($0 !~ /^[[:space:]]*$/ && $0 !~ /^Title:/ && $0 !~ /^URL:/ && $0 !~ /^Source:/) cnt++
            }
            END { if (cnt==0) exit 0; else exit 1 }
        ' "$file"; then
            rm -f -- "$file"
            echo "Deleted (Empty): $file"
            ((empty_deleted++))
        fi
    done < <(find "$HLTV_DIR" -type f -name "*.txt")
fi

# Удаление пустых файлов из Cybersport
if [ -d "$CYBERSPORT_DIR" ]; then
    echo "Scanning $CYBERSPORT_DIR for empty files..."
    while IFS= read -r file; do
        if awk '
            BEGIN { cnt = 0 }
            {
                if ($0 !~ /^[[:space:]]*$/ && $0 !~ /^Title:/ && $0 !~ /^URL:/ && $0 !~ /^Source:/ && $0 !~ /^Tag:/) cnt++
            }
            END { if (cnt==0) exit 0; else exit 1 }
        ' "$file"; then
            rm -f -- "$file"
            echo "Deleted (Empty): $file"
            ((empty_deleted++))
        fi
    done < <(find "$CYBERSPORT_DIR" -type f -name "*.txt")
fi

echo "Total empty deleted: $empty_deleted files"
echo ""
echo "=== Summary ==="
echo "Cloudflare blocks deleted: $deleted"
echo "Empty files deleted: $empty_deleted"
echo "Total deleted: $((deleted + empty_deleted))"