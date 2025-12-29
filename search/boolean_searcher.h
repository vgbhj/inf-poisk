#ifndef BOOLEAN_SEARCHER_H
#define BOOLEAN_SEARCHER_H

#include "boolean_index.h"
#include "query_parser.h"
#include "vector.h"
#include "string.h"

class BooleanSearcher {
private:
    BooleanIndex* index;

    Vector<unsigned int> evaluate_tokens(Vector<QueryToken>& tokens, size_t& pos) {
        Vector<unsigned int> result;

        if (pos >= tokens.get_size()) {
            return result;
        }

        QueryToken& token = tokens[pos];

        if (token.op == OP_TERM && strcmp(token.term.c_str(), "(") == 0) {
            pos++;
            result = evaluate_tokens(tokens, pos);
            if (pos < tokens.get_size() && strcmp(tokens[pos].term.c_str(), ")") == 0) {
                pos++;
            }
            return result;
        }

        if (token.op == OP_NOT) {
            pos++;
            if (pos < tokens.get_size() && tokens[pos].op == OP_TERM && strcmp(tokens[pos].term.c_str(), "(") != 0 && strcmp(tokens[pos].term.c_str(), ")") != 0) {
                result = index->search_not(tokens[pos].term);
                pos++;
            }
            return result;
        }

        if (token.op == OP_TERM) {
            PostingList* plist = index->get_posting_list(token.term.c_str());
            if (plist) {
                result = plist->get_docs();
            }
            pos++;
        }

        while (pos < tokens.get_size()) {
            QueryOperator op = tokens[pos].op;

            if (op == OP_AND) {
                pos++;
                Vector<unsigned int> right = evaluate_tokens(tokens, pos);
                Vector<unsigned int> new_result;

                for (size_t i = 0; i < result.get_size(); ++i) {
                    bool found = false;
                    for (size_t j = 0; j < right.get_size(); ++j) {
                        if (result[i] == right[j]) {
                            found = true;
                            break;
                        }
                    }
                    if (found) {
                        new_result.push_back(result[i]);
                    }
                }
                result = new_result;
            } else if (op == OP_OR) {
                pos++;
                Vector<unsigned int> right = evaluate_tokens(tokens, pos);
                Vector<unsigned int> new_result = result;

                for (size_t i = 0; i < right.get_size(); ++i) {
                    bool found = false;
                    for (size_t j = 0; j < new_result.get_size(); ++j) {
                        if (right[i] == new_result[j]) {
                            found = true;
                            break;
                        }
                    }
                    if (!found) {
                        new_result.push_back(right[i]);
                    }
                }

                for (size_t i = 0; i < new_result.get_size() - 1; ++i) {
                    for (size_t j = i + 1; j < new_result.get_size(); ++j) {
                        if (new_result[i] > new_result[j]) {
                            unsigned int temp = new_result[i];
                            new_result[i] = new_result[j];
                            new_result[j] = temp;
                        }
                    }
                }
                result = new_result;
            } else if (op == OP_NOT) {
                pos++;
                Vector<unsigned int> right = evaluate_tokens(tokens, pos);
                Vector<unsigned int> new_result;

                for (size_t i = 0; i < result.get_size(); ++i) {
                    bool found = false;
                    for (size_t j = 0; j < right.get_size(); ++j) {
                        if (result[i] == right[j]) {
                            found = true;
                            break;
                        }
                    }
                    if (!found) {
                        new_result.push_back(result[i]);
                    }
                }
                result = new_result;
            } else if (strcmp(tokens[pos].term.c_str(), ")") == 0) {
                break;
            } else {
                pos++;
            }
        }

        return result;
    }

public:
    BooleanSearcher(BooleanIndex* idx) : index(idx) {}

    Vector<unsigned int> search(const char* query_str) {
        QueryParser parser(query_str);
        Vector<QueryToken> tokens = parser.parse();

        if (tokens.get_size() == 0) {
            return Vector<unsigned int>();
        }

        size_t pos = 0;
        return evaluate_tokens(tokens, pos);
    }
};

#endif
