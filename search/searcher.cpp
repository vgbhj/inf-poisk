#include <cstdio>
#include <cstring>
#include "boolean_searcher.h"

int main(int argc, char* argv[]) {
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <index_file> [query]\n", argv[0]);
        fprintf(stderr, "Examples:\n");
        fprintf(stderr, "  %s index.bin \"word1 and word2\"\n", argv[0]);
        fprintf(stderr, "  %s index.bin \"word1 or word2\"\n", argv[0]);
        fprintf(stderr, "  %s index.bin \"-word1 word2\"\n", argv[0]);
        return 1;
    }

    const char* index_file = argv[1];

    BooleanIndex index;
    if (!index.load_index(index_file)) {
        fprintf(stderr, "Error loading index from: %s\n", index_file);
        return 1;
    }

    BooleanSearcher searcher(&index);

    if (argc >= 3) {
        const char* query = argv[2];
        Vector<unsigned int> results = searcher.search(query);

        printf("Query: %s\n", query);
        printf("Found documents: %zu\n", results.get_size());

        for (size_t i = 0; i < results.get_size(); ++i) {
            printf("Doc %u\n", results[i]);
        }
    } else {
        char query[2048];
        printf("Boolean Search Engine (quit to exit)\n");

        while (true) {
            printf("query> ");
            if (!fgets(query, sizeof(query), stdin)) break;

            size_t len = strlen(query);
            if (len > 0 && query[len - 1] == '\n') {
                query[len - 1] = '\0';
            }

            if (strcmp(query, "quit") == 0 || strcmp(query, "exit") == 0) {
                break;
            }

            Vector<unsigned int> results = searcher.search(query);
            printf("Found: %zu documents\n", results.get_size());

            for (size_t i = 0; i < results.get_size() && i < 10; ++i) {
                printf("  Doc %u\n", results[i]);
            }

            if (results.get_size() > 10) {
                printf("  ... and %zu more\n", results.get_size() - 10);
            }
        }
    }

    return 0;
}
