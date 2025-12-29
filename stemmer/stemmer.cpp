#include <iostream>
#include <fstream>
#include <filesystem>
#include <chrono>
#include <iomanip>

namespace fs = std::filesystem;

size_t my_strlen(const char* s) {
    size_t len = 0;
    while (s[len]) len++;
    return len;
}

void to_lower_utf8(char* s) {
    for (size_t i = 0; s[i]; ++i) {
        unsigned char c1 = (unsigned char)s[i];
        if (c1 == 0xD0) {
            if (s[i + 1]) {
                unsigned char c2 = (unsigned char)s[i + 1];
                if (c2 >= 0x90 && c2 <= 0xAF) s[i + 1] = c2 + 0x20;
                else if (c2 == 0x81) { s[i] = 0xD1; s[i + 1] = 0x91; }
                i++;
            }
        } else if (c1 >= 'A' && c1 <= 'Z') {
            s[i] = c1 + 32;
        }
    }
}

class PorterStemmer {
private:
    bool is_vowel(const char* word, size_t i) {
        unsigned char c1 = (unsigned char)word[i];
        if (!word[i+1]) return false;
        unsigned char c2 = (unsigned char)word[i + 1];
        if (c1 == 0xD0) {
            return (c2 == 0x90 || c2 == 0xB0 || c2 == 0x95 || c2 == 0xB5 || 
                    c2 == 0x98 || c2 == 0xB8 || c2 == 0x9E || c2 == 0xBE || 
                    c2 == 0xA3 || c2 == 0xC3 || c2 == 0xAB || c2 == 0xCB || 
                    c2 == 0xAD || c2 == 0xCD);
        }
        if (c1 == 0xD1) {
            return (c2 == 0x8E || c2 == 0x9E || c2 == 0x8F || c2 == 0x9F || 
                    c2 == 0x91 || c2 == 0x81);
        }
        return false;
    }

    bool ends_with(const char* word, size_t len, const char* suffix) {
        size_t s_len = my_strlen(suffix);
        if (s_len > len) return false;
        for (size_t i = 0; i < s_len; i++) {
            if (word[len - s_len + i] != suffix[i]) return false;
        }
        return true;
    }

    bool replace(char* word, size_t& len, const char* suffix, const char* repl = "") {
        if (ends_with(word, len, suffix)) {
            size_t s_len = my_strlen(suffix);
            size_t r_len = my_strlen(repl);
            len -= s_len;
            for (size_t i = 0; i < r_len; i++) word[len + i] = repl[i];
            len += r_len;
            word[len] = '\0';
            return true;
        }
        return false;
    }

    size_t find_rv(const char* word, size_t len) {
        for (size_t i = 0; i < len - 1; i += 2) {
            if (is_vowel(word, i)) return (i + 2 < len) ? i + 2 : len;
        }
        return len;
    }

public:
    void stem(char* word) {
        size_t len = my_strlen(word);
        if (len < 6) return;
        size_t rv_pos = find_rv(word, len);
        if (rv_pos >= len) return;
        char* rv = word + rv_pos;
        size_t r_len = len - rv_pos;

        bool changed = false;
        static const char* perf[] = {"ившись", "ывшись", "вшись", "ив", "ыв", "в"};
        for (const char* s : perf) if (replace(rv, r_len, s)) { changed = true; break; }

        if (!changed) {
            static const char* refl[] = {"ся", "сь"};
            for (const char* s : refl) replace(rv, r_len, s);

            static const char* adj[] = {"ее", "ие", "ые", "ое", "ими", "ыми", "ей", "ий", "ый", "ой", "ем", "им", "ым", "ом", "его", "ого", "ему", "ому", "их", "ых", "ую", "юю", "ая", "яя", "ою", "ею"};
            bool is_adj = false;
            for (const char* s : adj) if (replace(rv, r_len, s)) { is_adj = true; break; }

            if (is_adj) {
                static const char* part[] = {"ивш", "ывш", "ующ", "ем", "нн", "вш", "ющ", "щ"};
                for (const char* s : part) if (replace(rv, r_len, s)) break;
            } else {
                static const char* verb[] = {"ила", "ыла", "ена", "ейте", "уйте", "ите", "или", "ыли", "ей", "уй", "ил", "ыл", "им", "ым", "ен", "ят", "ует", "уют", "ит", "ыт", "ены", "ить", "ыть", "ишь", "ую", "ю"};
                bool is_verb = false;
                for (const char* s : verb) if (replace(rv, r_len, s)) { is_verb = true; break; }
                
                if (!is_verb) {
                    static const char* noun[] = {"иями", "ями", "ами", "ией", "иям", "ием", "ию", "ий", "ия", "ие", "ям", "ем", "ам", "ом", "ях", "ах", "ю", "ь", "и", "я", "а", "е"};
                    for (const char* s : noun) if (replace(rv, r_len, s)) break;
                }
            }
        }
        replace(rv, r_len, "и");
        static const char* deriv[] = {"ость", "ост"};
        for (const char* s : deriv) if (replace(rv, r_len, s)) break;

        if (!replace(rv, r_len, "ь")) {
            replace(rv, r_len, "ейше");
            replace(rv, r_len, "нн", "н");
        }

        // Фикс UTF-8: если после всех замен в конце остался "битый" байт (0xD0/0xD1)
        size_t final_len = rv_pos + r_len;
        if (final_len > 0) {
            unsigned char last = (unsigned char)word[final_len - 1];
            if (last == 0xD0 || last == 0xD1) {
                word[final_len - 1] = '\0';
            } else {
                word[final_len] = '\0';
            }
        }
    }
};

class Processor {
private:
    size_t total_tokens = 0;
    size_t total_chars = 0;
    PorterStemmer stemmer;

    bool is_word_char(unsigned char c) {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c > 127 || c == '-' || c == '_';
    }

public:
    void process_file(const char* in_path, std::ofstream& out_file) {
        std::ifstream file(in_path, std::ios::binary);
        if (!file) return;
        char word_buf[1024];
        size_t w_pos = 0;
        char c;
        while (file.get(c)) {
            if (is_word_char((unsigned char)c)) {
                if (w_pos < 1023) word_buf[w_pos++] = c;
            } else {
                if (w_pos > 0) {
                    word_buf[w_pos] = '\0';
                    to_lower_utf8(word_buf);
                    stemmer.stem(word_buf);
                    size_t slen = my_strlen(word_buf);
                    if (slen > 0) {
                        out_file << word_buf << "\n";
                        total_tokens++;
                        total_chars += slen;
                    }
                    w_pos = 0;
                }
            }
        }
        if (w_pos > 0) { // Обработка последнего слова, если файл не кончается разделителем
            word_buf[w_pos] = '\0';
            to_lower_utf8(word_buf);
            stemmer.stem(word_buf);
            if (my_strlen(word_buf) > 0) {
                out_file << word_buf << "\n";
                total_tokens++;
                total_chars += my_strlen(word_buf);
            }
        }
    }

    void process_dir(const fs::path& root, std::ofstream& out_file) {
        for (const auto& entry : fs::recursive_directory_iterator(root)) {
            if (entry.is_regular_file() && entry.path().extension() == ".txt") {
                process_file(entry.path().string().c_str(), out_file);
            }
        }
    }

    size_t get_count() { return total_tokens; }
    double get_avg() { return total_tokens == 0 ? 0 : (double)total_chars / total_tokens; }
};

int main(int argc, char* argv[]) {
    if (argc < 3) return 1;
    const char* input = argv[1];
    const char* output_tokens = argv[2];
    const char* output_stats = (argc > 3) ? argv[3] : "stats.txt";

    std::ofstream out(output_tokens);
    Processor proc;
    
    auto start = std::chrono::high_resolution_clock::now();
    if (fs::is_directory(input)) proc.process_dir(input, out);
    else proc.process_file(input, out);
    auto end = std::chrono::high_resolution_clock::now();
    
    out.close();
    std::ofstream stats(output_stats);
    stats << "Total tokens: " << proc.get_count() << "\n";
    stats << "Average length: " << std::fixed << std::setprecision(2) << proc.get_avg() << "\n";
    
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(end - start).count();
    std::cout << "Time: " << ms << " ms\n";
    std::cout << "Tokens: " << proc.get_count() << "\n";
    std::cout << "Avg Length: " << std::fixed << std::setprecision(2) << proc.get_avg() << "\n";

    return 0;
}