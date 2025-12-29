#ifndef BOOLEAN_INDEX_H
#define BOOLEAN_INDEX_H

#include "hashmap.h"
#include "string.h"
#include "posting_list.h"
#include "vector.h"
#include <cstdio>
#include <cstring>

class BooleanIndex {
private:
    HashMap<String, PostingList> index;
    HashMap<unsigned int, String> documents;
    unsigned int nextDocId;

    void to_lower(char* str) {
        for (int i = 0; str[i]; ++i) {
            if (str[i] >= 'A' && str[i] <= 'Z') {
                str[i] += 32;
            }
        }
    }

    void tokenize_and_index(unsigned int docId, const char* text) {
        char buffer[4096];
        int bufIdx = 0;

        for (int i = 0; text[i]; ++i) {
            unsigned char c = text[i];
            bool is_word_char = (c >= '0' && c <= '9') || 
                               (c >= 'a' && c <= 'z') || 
                               (c >= 'A' && c <= 'Z') || 
                               c == '-' || c == '_' || c > 127;

            if (is_word_char) {
                if (bufIdx < 4095) {
                    buffer[bufIdx++] = c;
                }
            } else {
                if (bufIdx > 0) {
                    buffer[bufIdx] = '\0';
                    to_lower(buffer);
                    String token(buffer);
                    index[token].add_doc(docId);
                    bufIdx = 0;
                }
            }
        }

        if (bufIdx > 0) {
            buffer[bufIdx] = '\0';
            to_lower(buffer);
            String token(buffer);
            index[token].add_doc(docId);
        }
    }

public:
    BooleanIndex() : nextDocId(0) {}

    unsigned int add_document(const char* text) {
        unsigned int docId = nextDocId++;
        documents[docId] = String(text);
        tokenize_and_index(docId, text);
        return docId;
    }

    PostingList* get_posting_list(const char* term) {
        String key(term);
        return index.get(key);
    }

    Vector<unsigned int> search_and(const Vector<String>& terms) {
        if (terms.get_size() == 0) {
            return Vector<unsigned int>();
        }

        PostingList* result = index.get(terms[0]);
        Vector<unsigned int> current_result;

        if (result) {
            current_result = result->get_docs();
        }

        for (size_t i = 1; i < terms.get_size(); ++i) {
            PostingList* plist = index.get(terms[i]);
            Vector<unsigned int> next_result;

            if (plist) {
                for (size_t j = 0; j < current_result.get_size(); ++j) {
                    if (plist->contains(current_result[j])) {
                        next_result.push_back(current_result[j]);
                    }
                }
            }

            current_result = next_result;
        }

        return current_result;
    }

    Vector<unsigned int> search_or(const Vector<String>& terms) {
        Vector<unsigned int> result;

        for (size_t i = 0; i < terms.get_size(); ++i) {
            PostingList* plist = index.get(terms[i]);
            if (plist) {
                Vector<unsigned int> docs = plist->get_docs();
                for (size_t j = 0; j < docs.get_size(); ++j) {
                    bool found = false;
                    for (size_t k = 0; k < result.get_size(); ++k) {
                        if (result[k] == docs[j]) {
                            found = true;
                            break;
                        }
                    }
                    if (!found) {
                        result.push_back(docs[j]);
                    }
                }
            }
        }

        for (size_t i = 0; i < result.get_size() - 1; ++i) {
            for (size_t j = i + 1; j < result.get_size(); ++j) {
                if (result[i] > result[j]) {
                    unsigned int temp = result[i];
                    result[i] = result[j];
                    result[j] = temp;
                }
            }
        }

        return result;
    }

    Vector<unsigned int> search_not(const String& term) {
        Vector<unsigned int> result;
        PostingList* plist = index.get(term);

        for (unsigned int i = 0; i < nextDocId; ++i) {
            if (!plist || !plist->contains(i)) {
                result.push_back(i);
            }
        }

        return result;
    }

    const char* get_document_text(unsigned int docId) {
        String* doc = documents.get(docId);
        return doc ? doc->c_str() : nullptr;
    }

    size_t get_total_documents() const {
        return nextDocId;
    }

    size_t get_total_terms() const {
        return index.size();
    }

    Vector<String> keys() const {
        return index.keys();
    }

    void save_index(const char* filename) {
        FILE* f = fopen(filename, "w");
        if (!f) return;

        fprintf(f, "BOOLEAN_INDEX\n");
        fprintf(f, "%u\n", nextDocId);
        fprintf(f, "%zu\n", index.size());

        Vector<String> terms = index.keys();
        for (size_t i = 0; i < terms.get_size(); ++i) {
            PostingList* plist = index.get(terms[i]);
            if (plist) {
                fprintf(f, "%s:", terms[i].c_str());
                Vector<unsigned int> docs = plist->get_docs();
                for (size_t j = 0; j < docs.get_size(); ++j) {
                    fprintf(f, " %u", docs[j]);
                }
                fprintf(f, "\n");
            }
        }

        fclose(f);
    }

    bool load_index(const char* filename) {
        FILE* f = fopen(filename, "r");
        if (!f) return false;

        char header[32];
        if (fscanf(f, "%s", header) != 1 || strcmp(header, "BOOLEAN_INDEX") != 0) {
            fclose(f);
            return false;
        }

        size_t docCount, termCount;
        if (fscanf(f, "%zu %zu", &docCount, &termCount) != 2) {
            fclose(f);
            return false;
        }

        nextDocId = docCount;

        char line[8192];
        fgets(line, sizeof(line), f);

        for (size_t i = 0; i < termCount; ++i) {
            if (!fgets(line, sizeof(line), f)) break;

            char* colon = strchr(line, ':');
            if (!colon) continue;

            *colon = '\0';
            String term(line);

            unsigned int docId;
            char* ptr = colon + 1;
            while (sscanf(ptr, "%u", &docId) == 1) {
                index[term].add_doc(docId);
                char* space = strchr(ptr, ' ');
                if (!space) break;
                ptr = space + 1;
            }
        }

        fclose(f);
        return true;
    }
};

#endif
