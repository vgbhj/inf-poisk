#include <cstdio>
#include <cstring>
#include <dirent.h>
#include "boolean_index.h"

int main(int argc, char* argv[]) {
    if (argc < 3) {
        fprintf(stderr, "Usage: %s <corpus_dir> <index_file>\n", argv[0]);
        return 1;
    }

    const char* corpus_dir = argv[1];
    const char* index_file = argv[2];

    BooleanIndex index;

    DIR* dir = opendir(corpus_dir);
    if (!dir) {
        fprintf(stderr, "Error opening directory: %s\n", corpus_dir);
        return 1;
    }

    struct dirent* entry;
    unsigned int doc_count = 0;

    while ((entry = readdir(dir)) != nullptr) {
        if (entry->d_type != DT_REG) continue;

        char filepath[512];
        snprintf(filepath, sizeof(filepath), "%s/%s", corpus_dir, entry->d_name);

        FILE* f = fopen(filepath, "r");
        if (!f) continue;

        char buffer[65536];
        size_t bytes_read = fread(buffer, 1, sizeof(buffer) - 1, f);
        buffer[bytes_read] = '\0';
        fclose(f);

        index.add_document(buffer);
        doc_count++;

        if (doc_count % 100 == 0) {
            fprintf(stderr, "Indexed %u documents\n", doc_count);
        }
    }

    closedir(dir);

    index.save_index(index_file);

    printf("Total documents: %u\n", doc_count);
    printf("Total terms: %zu\n", index.get_total_terms());
    printf("Index saved to: %s\n", index_file);

    return 0;
}
