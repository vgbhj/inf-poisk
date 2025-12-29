#ifndef POSTING_LIST_H
#define POSTING_LIST_H

#include "vector.h"

class PostingList {
private:
    Vector<unsigned int> docIds;

public:
    PostingList() {}

    void add_doc(unsigned int docId) {
        for (size_t i = 0; i < docIds.get_size(); ++i) {
            if (docIds[i] == docId) {
                return;
            }
        }
        docIds.push_back(docId);
        sort_docs();
    }

    Vector<unsigned int> get_docs() const {
        return docIds;
    }

    size_t size() const {
        return docIds.get_size();
    }

    bool contains(unsigned int docId) const {
        for (size_t i = 0; i < docIds.get_size(); ++i) {
            if (docIds[i] == docId) {
                return true;
            }
        }
        return false;
    }

    Vector<unsigned int> intersect(const PostingList& other) const {
        Vector<unsigned int> result;
        for (size_t i = 0; i < docIds.get_size(); ++i) {
            if (other.contains(docIds[i])) {
                result.push_back(docIds[i]);
            }
        }
        return result;
    }

    Vector<unsigned int> unite(const PostingList& other) const {
        Vector<unsigned int> result = docIds;
        for (size_t i = 0; i < other.docIds.get_size(); ++i) {
            bool found = false;
            for (size_t j = 0; j < result.get_size(); ++j) {
                if (result[j] == other.docIds[i]) {
                    found = true;
                    break;
                }
            }
            if (!found) {
                result.push_back(other.docIds[i]);
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

    Vector<unsigned int> difference(const PostingList& other) const {
        Vector<unsigned int> result;
        for (size_t i = 0; i < docIds.get_size(); ++i) {
            if (!other.contains(docIds[i])) {
                result.push_back(docIds[i]);
            }
        }
        return result;
    }

private:
    void sort_docs() {
        for (size_t i = 0; i < docIds.get_size() - 1; ++i) {
            for (size_t j = i + 1; j < docIds.get_size(); ++j) {
                if (docIds[i] > docIds[j]) {
                    unsigned int temp = docIds[i];
                    docIds[i] = docIds[j];
                    docIds[j] = temp;
                }
            }
        }
    }
};

#endif
