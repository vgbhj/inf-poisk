#ifndef QUERY_PARSER_H
#define QUERY_PARSER_H

#include "vector.h"
#include "string.h"

enum QueryOperator {
    OP_AND,
    OP_OR,
    OP_NOT,
    OP_TERM
};

class QueryToken {
public:
    QueryOperator op;
    String term;

    QueryToken() : op(OP_TERM) {}
    QueryToken(QueryOperator o) : op(o) {}
    QueryToken(QueryOperator o, const char* t) : op(o), term(t) {}
};

class QueryParser {
private:
    const char* query;
    size_t pos;

    void skip_spaces() {
        while (query[pos] && query[pos] == ' ') {
            pos++;
        }
    }

    void to_lower_str(char* str) {
        for (int i = 0; str[i]; ++i) {
            if (str[i] >= 'A' && str[i] <= 'Z') {
                str[i] += 32;
            }
        }
    }

public:
    QueryParser(const char* q) : query(q), pos(0) {}

    Vector<QueryToken> parse() {
        Vector<QueryToken> tokens;
        skip_spaces();

        while (query[pos]) {
            skip_spaces();

            if (!query[pos]) break;

            if (query[pos] == '(') {
                QueryToken t(OP_TERM, "(");
                tokens.push_back(t);
                pos++;
            } else if (query[pos] == ')') {
                QueryToken t(OP_TERM, ")");
                tokens.push_back(t);
                pos++;
            } else if (query[pos] == '-') {
                pos++;
                skip_spaces();
                char term_buf[256];
                int i = 0;
                while (query[pos] && query[pos] != ' ' && query[pos] != '(' && query[pos] != ')' && i < 255) {
                    term_buf[i++] = query[pos++];
                }
                term_buf[i] = '\0';
                to_lower_str(term_buf);
                QueryToken t(OP_NOT, term_buf);
                tokens.push_back(t);
            } else {
                char word[256];
                int i = 0;
                while (query[pos] && query[pos] != ' ' && query[pos] != '(' && query[pos] != ')' && i < 255) {
                    word[i++] = query[pos++];
                }
                word[i] = '\0';
                to_lower_str(word);

                if (strcmp(word, "and") == 0) {
                    QueryToken t(OP_AND);
                    tokens.push_back(t);
                } else if (strcmp(word, "or") == 0) {
                    QueryToken t(OP_OR);
                    tokens.push_back(t);
                } else if (strcmp(word, "not") == 0) {
                    QueryToken t(OP_NOT);
                    tokens.push_back(t);
                } else {
                    QueryToken t(OP_TERM, word);
                    tokens.push_back(t);
                }
            }
        }

        return tokens;
    }
};

#endif
