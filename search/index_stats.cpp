#include <cstdio>
#include <cstring>
#include "boolean_index.h"

int main(int argc, char* argv[]) {
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <index_file>\n", argv[0]);
        return 1;
    }

    const char* index_file = argv[1];

    BooleanIndex index;
    if (!index.load_index(index_file)) {
        fprintf(stderr, "Error loading index from: %s\n", index_file);
        return 1;
    }

    printf("=== Index Statistics ===\n");
    printf("Total documents: %zu\n", index.get_total_documents());
    printf("Total unique terms: %zu\n", index.get_total_terms());
    printf("\n");

    if (index.get_total_terms() > 0) {
        printf("=== Term Statistics ===\n");
        
        size_t total_posting_size = 0;
        size_t max_posting_size = 0;
        size_t min_posting_size = index.get_total_documents() + 1;
        
        printf("\nMost frequent terms (top 20):\n");
        printf("Rank | Term | Document Count\n");
        printf("-----|------|----------------\n");

        int rank = 1;
        Vector<String> terms = index.keys();
        
        for (size_t i = 0; i < terms.get_size() && rank <= 20; ++i) {
            PostingList* plist = index.get_posting_list(terms[i].c_str());
            if (plist) {
                size_t count = plist->size();
                total_posting_size += count;
                if (count > max_posting_size) max_posting_size = count;
                if (count < min_posting_size) min_posting_size = count;
                
                printf("%4d | %s | %lu\n", rank++, terms[i].c_str(), count);
            }
        }

        printf("\n=== Index Characteristics ===\n");
        printf("Average posting list size: %.2f\n", 
               index.get_total_terms() > 0 ? 
               (double)total_posting_size / index.get_total_terms() : 0.0);
        printf("Max posting list size: %zu\n", max_posting_size);
        printf("Min posting list size: %zu\n", min_posting_size > index.get_total_documents() ? 0 : min_posting_size);
    }

    return 0;
}
