#include <iostream>
#include <fstream>
#include <cmath>
#include <iomanip>

struct Node {
    char* term;
    size_t count;
    Node* next;
};

class FrequencyCounter {
private:
    static const size_t TABLE_SIZE = 131072;
    Node* table[TABLE_SIZE];
    size_t unique_count = 0;
    size_t total_tokens = 0;

    size_t hash(const char* s) {
        size_t h = 5381;
        int c;
        while ((c = *s++)) h = ((h << 5) + h) + c;
        return h % TABLE_SIZE;
    }

    char* my_strdup(const char* s) {
        size_t len = 0;
        while (s[len]) len++;
        char* res = new char[len + 1];
        for (size_t i = 0; i <= len; i++) res[i] = s[i];
        return res;
    }

    bool my_strcmp(const char* s1, const char* s2) {
        while (*s1 && (*s1 == *s2)) { s1++; s2++; }
        return *(unsigned char*)s1 == *(unsigned char*)s2;
    }

public:
    FrequencyCounter() {
        for (size_t i = 0; i < TABLE_SIZE; i++) table[i] = nullptr;
    }

    ~FrequencyCounter() {
        for (size_t i = 0; i < TABLE_SIZE; i++) {
            Node* curr = table[i];
            while (curr) {
                Node* tmp = curr;
                curr = curr->next;
                delete[] tmp->term;
                delete tmp;
            }
        }
    }

    void add(const char* term) {
        if (!term || !term[0]) return;
        total_tokens++;
        size_t h = hash(term);
        Node* curr = table[h];
        while (curr) {
            if (my_strcmp(curr->term, term)) {
                curr->count++;
                return;
            }
            curr = curr->next;
        }
        Node* newNode = new Node;
        newNode->term = my_strdup(term);
        newNode->count = 1;
        newNode->next = table[h];
        table[h] = newNode;
        unique_count++;
    }

    void sort_and_save(const char* filename) {
        Node** flat_list = new Node*[unique_count];
        size_t idx = 0;
        for (size_t i = 0; i < TABLE_SIZE; i++) {
            Node* curr = table[i];
            while (curr) {
                flat_list[idx++] = curr;
                curr = curr->next;
            }
        }

        for (size_t i = 0; i < unique_count - 1; i++) {
            for (size_t j = 0; j < unique_count - i - 1; j++) {
                if (flat_list[j]->count < flat_list[j + 1]->count) {
                    Node* temp = flat_list[j];
                    flat_list[j] = flat_list[j + 1];
                    flat_list[j + 1] = temp;
                }
            }
        }

        std::ofstream out(filename);
        out << "rank\tterm\tfrequency\tlog_rank\tlog_frequency\n";
        for (size_t i = 0; i < unique_count; i++) {
            size_t rank = i + 1;
            out << rank << "\t" << flat_list[i]->term << "\t" << flat_list[i]->count << "\t"
                << std::fixed << std::setprecision(4) << std::log10((double)rank) << "\t"
                << std::log10((double)flat_list[i]->count) << "\n";
        }
        delete[] flat_list;
    }

    size_t get_unique() { return unique_count; }
    size_t get_total() { return total_tokens; }
};

int main(int argc, char* argv[]) {
    if (argc < 3) return 1;
    std::ifstream input(argv[1]);
    if (!input) return 1;

    FrequencyCounter fc;
    char buffer[1024];
    while (input >> buffer) {
        fc.add(buffer);
    }
    input.close();

    fc.sort_and_save(argv[2]);

    std::cout << "Unique: " << fc.get_unique() << "\n";
    std::cout << "Total: " << fc.get_total() << "\n";

    return 0;
}