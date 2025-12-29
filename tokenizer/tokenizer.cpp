#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <algorithm>
#include <cctype>
#include <chrono>
#include <iomanip>
#include <filesystem>

namespace fs = std::filesystem;

class Tokenizer {
private:
    size_t totalTokens = 0;
    size_t totalChars = 0;
    std::vector<std::string> tokens;

    void toLowerUtf8(std::string& s) {
        for (size_t i = 0; i < s.length(); ++i) {
            unsigned char c1 = static_cast<unsigned char>(s[i]);
            if (c1 == 0xD0) {
                if (i + 1 < s.length()) {
                    unsigned char c2 = static_cast<unsigned char>(s[i + 1]);
                    if (c2 >= 0x90 && c2 <= 0xAF) {
                        s[i + 1] = c2 + 0x20;
                    } else if (c2 == 0x81) {
                        s[i] = 0xD1;
                        s[i + 1] = 0x91;
                    }
                }
            } else if (c1 == 0xD1 && i + 1 < s.length()) {
                unsigned char c2 = static_cast<unsigned char>(s[i + 1]);
                if (c2 == 0x90) {
                    s[i] = 0xD0;
                    s[i + 1] = 0x81;
                }
            } else {
                s[i] = static_cast<char>(std::tolower(c1));
            }
        }
    }

    bool isWordChar(unsigned char c) {
        return std::isalnum(c) || c > 127 || c == '-' || c == '_';
    }

public:
    void processFile(const std::string& filepath) {
        std::ifstream file(filepath, std::ios::binary);
        if (!file.is_open()) return;

        std::string word;
        char c;
        while (file.get(c)) {
            unsigned char uc = static_cast<unsigned char>(c);
            if (isWordChar(uc)) {
                word += c;
            } else {
                if (!word.empty()) {
                    toLowerUtf8(word);
                    tokens.push_back(word);
                    totalChars += word.length();
                    totalTokens++;
                    word.clear();
                }
            }
        }
        if (!word.empty()) {
            toLowerUtf8(word);
            tokens.push_back(word);
            totalChars += word.length();
            totalTokens++;
        }
    }

    void processDirectory(const std::string& dirpath) {
        for (const auto& entry : fs::recursive_directory_iterator(dirpath)) {
            if (entry.is_regular_file() && entry.path().extension() == ".txt") {
                processFile(entry.path().string());
            }
        }
    }

    double getAverageLength() const {
        return totalTokens == 0 ? 0.0 : static_cast<double>(totalChars) / totalTokens;
    }

    void saveResults(const std::string& outFile, const std::string& statFile) {
        std::ofstream out(outFile);
        for (const auto& t : tokens) out << t << "\n";
        
        std::ofstream stat(statFile);
        stat << "Total tokens: " << totalTokens << "\n";
        stat << "Average length: " << std::fixed << std::setprecision(2) << getAverageLength() << "\n";
    }

    size_t getTotalTokens() const { return totalTokens; }
};

int main(int argc, char* argv[]) {
    if (argc < 3) return 1;

    std::string input = argv[1];
    std::string output = argv[2];
    std::string stats = (argc > 3) ? argv[3] : "stats.txt";

    Tokenizer tokenizer;
    auto start = std::chrono::high_resolution_clock::now();

    if (fs::is_directory(input)) tokenizer.processDirectory(input);
    else if (fs::is_regular_file(input)) tokenizer.processFile(input);

    auto end = std::chrono::high_resolution_clock::now();
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(end - start).count();

    tokenizer.saveResults(output, stats);

    std::cout << "Time: " << ms << " ms" << std::endl;
    std::cout << "Tokens: " << tokenizer.getTotalTokens() << std::endl;
    std::cout << "Avg Length: " << std::fixed << std::setprecision(2) << tokenizer.getAverageLength() << std::endl;

    return 0;
}