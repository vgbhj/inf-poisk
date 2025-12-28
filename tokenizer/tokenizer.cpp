#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <sstream>
#include <algorithm>
#include <cctype>
#include <chrono>
#include <iomanip>
#include <filesystem>

namespace fs = std::filesystem;

class Tokenizer {
private:
    std::vector<std::string> tokens;
    size_t totalChars = 0;

    bool isWordChar(char c) {
        return std::isalnum(c) || c == '-' || c == '_' || (c >= 128);
    }

    bool isPunctuation(char c) {
        return std::ispunct(c) && c != '-' && c != '_';
    }

    void tokenizeText(const std::string& text) {
        std::string currentToken;
        bool inWord = false;

        for (size_t i = 0; i < text.length(); ++i) {
            char c = text[i];
            
            if (std::isspace(c)) {
                if (inWord && !currentToken.empty()) {
                    tokens.push_back(currentToken);
                    totalChars += currentToken.length();
                    currentToken.clear();
                    inWord = false;
                }
            } else if (isWordChar(c)) {
                currentToken += std::tolower(c);
                inWord = true;
            } else if (isPunctuation(c)) {
                if (inWord && !currentToken.empty()) {
                    tokens.push_back(currentToken);
                    totalChars += currentToken.length();
                    currentToken.clear();
                    inWord = false;
                }
                if (c == '.' || c == '!' || c == '?') {
                    std::string punct(1, c);
                    tokens.push_back(punct);
                    totalChars += 1;
                }
            } else {
                if (inWord && !currentToken.empty()) {
                    tokens.push_back(currentToken);
                    totalChars += currentToken.length();
                    currentToken.clear();
                    inWord = false;
                }
            }
        }

        if (!currentToken.empty()) {
            tokens.push_back(currentToken);
            totalChars += currentToken.length();
        }
    }

public:
    void processFile(const std::string& filepath) {
        std::ifstream file(filepath);
        if (!file.is_open()) {
            std::cerr << "Error opening file: " << filepath << std::endl;
            return;
        }

        std::string line;
        std::string text;
        while (std::getline(file, line)) {
            text += line + " ";
        }
        file.close();

        tokenizeText(text);
    }

    void processDirectory(const std::string& dirpath) {
        size_t fileCount = 0;
        for (const auto& entry : fs::directory_iterator(dirpath)) {
            if (entry.is_regular_file() && entry.path().extension() == ".txt") {
                processFile(entry.path().string());
                fileCount++;
                if (fileCount % 100 == 0) {
                    std::cerr << "Processed " << fileCount << " files, tokens: " << tokens.size() << std::endl;
                }
            }
        }
    }

    size_t getTokenCount() const {
        return tokens.size();
    }

    double getAverageLength() const {
        if (tokens.empty()) return 0.0;
        return static_cast<double>(totalChars) / tokens.size();
    }

    void saveTokens(const std::string& outputFile) {
        std::ofstream out(outputFile);
        for (const auto& token : tokens) {
            out << token << "\n";
        }
        out.close();
    }

    void saveStatistics(const std::string& statsFile) {
        std::ofstream out(statsFile);
        out << "Total tokens: " << tokens.size() << "\n";
        out << "Average token length: " << std::fixed << std::setprecision(2) << getAverageLength() << "\n";
        out.close();
    }
};

int main(int argc, char* argv[]) {
    if (argc < 3) {
        std::cerr << "Usage: " << argv[0] << " <input_dir> <output_file> [stats_file]" << std::endl;
        return 1;
    }

    std::string inputDir = argv[1];
    std::string outputFile = argv[2];
    std::string statsFile = (argc > 3) ? argv[3] : "tokenizer_stats.txt";

    auto start = std::chrono::high_resolution_clock::now();

    Tokenizer tokenizer;
    
    if (fs::is_directory(inputDir)) {
        tokenizer.processDirectory(inputDir);
    } else if (fs::is_regular_file(inputDir)) {
        tokenizer.processFile(inputDir);
    } else {
        std::cerr << "Error: " << inputDir << " is not a valid file or directory" << std::endl;
        return 1;
    }

    auto end = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);

    tokenizer.saveTokens(outputFile);
    tokenizer.saveStatistics(statsFile);

    std::cout << "Tokenization completed in " << duration.count() << " ms" << std::endl;
    std::cout << "Total tokens: " << tokenizer.getTokenCount() << std::endl;
    std::cout << "Average token length: " << std::fixed << std::setprecision(2) 
              << tokenizer.getAverageLength() << std::endl;

    return 0;
}

