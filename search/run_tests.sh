#!/bin/bash

set -e

echo "=== Boolean Search Engine - Build and Test ==="
echo ""

cd "$(dirname "$0")"

echo "1. Building the search engine..."
make clean
make
echo "✓ Build successful"
echo ""

echo "2. Building index from test documents..."
./index_builder test_docs test_index.idx
echo ""

echo "3. Displaying index statistics..."
./index_stats test_index.idx
echo ""

echo "4. Running sample queries..."
echo ""

echo "Query 1: fox"
./searcher test_index.idx "fox"
echo ""

echo "Query 2: fox and animals"
./searcher test_index.idx "fox and animals"
echo ""

echo "Query 3: animals or learning"
./searcher test_index.idx "animals or learning"
echo ""

echo "Query 4: database -forest"
./searcher test_index.idx "database -forest"
echo ""

echo "Query 5: (fox or bear) and animals"
./searcher test_index.idx "(fox or bear) and animals"
echo ""

echo "=== All tests completed successfully ==="
