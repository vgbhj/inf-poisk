#include <iostream>
#include <fstream>
#include <string>
#include <vector>
#include <algorithm>
#include <cctype>

class PorterStemmer {
private:
    bool isVowel(char c, const std::string& word, size_t pos) {
        if (pos >= word.length()) return false;
        char ch = std::tolower(word[pos]);
        return ch == 'a' || ch == 'e' || ch == 'i' || ch == 'o' || ch == 'u' || 
               (ch == 'y' && pos > 0 && !isVowel(word[pos-1], word, pos-1));
    }

    bool hasVowel(const std::string& word) {
        for (size_t i = 0; i < word.length(); ++i) {
            if (isVowel(word[i], word, i)) return true;
        }
        return false;
    }

    size_t measure(const std::string& word) {
        size_t m = 0;
        bool inVowel = false;
        
        for (size_t i = 0; i < word.length(); ++i) {
            bool v = isVowel(word[i], word, i);
            if (!inVowel && v) {
                inVowel = true;
            } else if (inVowel && !v) {
                m++;
                inVowel = false;
            }
        }
        return m;
    }

    bool endsWith(const std::string& word, const std::string& suffix) {
        if (word.length() < suffix.length()) return false;
        return word.substr(word.length() - suffix.length()) == suffix;
    }

    std::string replaceSuffix(std::string word, const std::string& suffix, const std::string& replacement) {
        if (endsWith(word, suffix)) {
            word = word.substr(0, word.length() - suffix.length()) + replacement;
        }
        return word;
    }

    bool step1a(std::string& word) {
        if (endsWith(word, "sses")) {
            word = replaceSuffix(word, "sses", "ss");
            return true;
        }
        if (endsWith(word, "ies")) {
            word = replaceSuffix(word, "ies", "i");
            return true;
        }
        if (endsWith(word, "ss")) {
            return false;
        }
        if (endsWith(word, "s")) {
            word = word.substr(0, word.length() - 1);
            return true;
        }
        return false;
    }

    bool step1b(std::string& word) {
        if (endsWith(word, "eed")) {
            std::string stem = word.substr(0, word.length() - 3);
            if (measure(stem) > 0) {
                word = stem + "ee";
                return true;
            }
            return false;
        }
        
        if ((endsWith(word, "ed") && hasVowel(word.substr(0, word.length() - 2))) ||
            (endsWith(word, "ing") && hasVowel(word.substr(0, word.length() - 3)))) {
            
            if (endsWith(word, "ed")) {
                word = word.substr(0, word.length() - 2);
            } else {
                word = word.substr(0, word.length() - 3);
            }
            
            if (endsWith(word, "at")) {
                word += "e";
            } else if (endsWith(word, "bl")) {
                word += "e";
            } else if (endsWith(word, "iz")) {
                word += "e";
            } else if (word.length() >= 2 && word[word.length()-1] == word[word.length()-2] &&
                       !endsWith(word, "l") && !endsWith(word, "s") && !endsWith(word, "z")) {
                word = word.substr(0, word.length() - 1);
            } else if (measure(word) == 1 && word.length() >= 3) {
                size_t last = word.length() - 1;
                if (!isVowel(word[last-1], word, last-1) && 
                    isVowel(word[last], word, last) && 
                    !isVowel(word[last+1], word, last+1) &&
                    word[last] != 'w' && word[last] != 'x' && word[last] != 'y') {
                    word += "e";
                }
            }
            return true;
        }
        return false;
    }

    bool step1c(std::string& word) {
        if (word.length() > 0 && word[word.length()-1] == 'y' && hasVowel(word.substr(0, word.length()-1))) {
            word[word.length()-1] = 'i';
            return true;
        }
        return false;
    }

    bool step2(std::string& word) {
        std::vector<std::pair<std::string, std::string>> rules = {
            {"ational", "ate"}, {"tional", "tion"}, {"enci", "ence"}, {"anci", "ance"},
            {"izer", "ize"}, {"abli", "able"}, {"alli", "al"}, {"entli", "ent"},
            {"eli", "e"}, {"ousli", "ous"}, {"ization", "ize"}, {"ation", "ate"},
            {"ator", "ate"}, {"alism", "al"}, {"iveness", "ive"}, {"fulness", "ful"},
            {"ousness", "ous"}, {"aliti", "al"}, {"iviti", "ive"}, {"biliti", "ble"},
            {"logi", "log"}
        };
        
        for (const auto& rule : rules) {
            if (endsWith(word, rule.first)) {
                std::string stem = word.substr(0, word.length() - rule.first.length());
                if (measure(stem) > 0) {
                    word = stem + rule.second;
                    return true;
                }
            }
        }
        return false;
    }

    bool step3(std::string& word) {
        std::vector<std::pair<std::string, std::string>> rules = {
            {"icate", "ic"}, {"ative", ""}, {"alize", "al"}, {"iciti", "ic"},
            {"ical", "ic"}, {"ful", ""}, {"ness", ""}
        };
        
        for (const auto& rule : rules) {
            if (endsWith(word, rule.first)) {
                std::string stem = word.substr(0, word.length() - rule.first.length());
                if (measure(stem) > 0) {
                    word = stem + rule.second;
                    return true;
                }
            }
        }
        return false;
    }

    bool step4(std::string& word) {
        std::vector<std::string> suffixes = {"al", "ance", "ence", "er", "ic", "able", "ible", 
                                              "ant", "ement", "ment", "ent", "ion", "ou", "ism", 
                                              "ate", "iti", "ous", "ive", "ize"};
        
        for (const auto& suffix : suffixes) {
            if (endsWith(word, suffix)) {
                if (suffix == "ion") {
                    if (word.length() > 3 && (word[word.length()-4] == 's' || word[word.length()-4] == 't')) {
                        std::string stem = word.substr(0, word.length() - 3);
                        if (measure(stem) > 1) {
                            word = stem;
                            return true;
                        }
                    }
                } else {
                    std::string stem = word.substr(0, word.length() - suffix.length());
                    if (measure(stem) > 1) {
                        word = stem;
                        return true;
                    }
                }
            }
        }
        return false;
    }

    bool step5a(std::string& word) {
        if (endsWith(word, "e")) {
            std::string stem = word.substr(0, word.length() - 1);
            if (measure(stem) > 1) {
                word = stem;
                return true;
            }
            if (measure(stem) == 1) {
                size_t last = stem.length() - 1;
                if (last >= 2 && !isVowel(stem[last-1], stem, last-1) && 
                    isVowel(stem[last], stem, last) && 
                    !isVowel(stem[last+1], stem, last+1) &&
                    stem[last] != 'w' && stem[last] != 'x' && stem[last] != 'y') {
                    return false;
                }
                word = stem;
                return true;
            }
        }
        return false;
    }

    bool step5b(std::string& word) {
        if (measure(word) > 1 && word.length() >= 2 && 
            word[word.length()-1] == word[word.length()-2] && word[word.length()-1] == 'l') {
            word = word.substr(0, word.length() - 1);
            return true;
        }
        return false;
    }

public:
    std::string stem(const std::string& word) {
        if (word.length() < 3) return word;
        
        std::string result = word;
        std::transform(result.begin(), result.end(), result.begin(), ::tolower);
        
        step1a(result);
        step1b(result);
        step1c(result);
        step2(result);
        step3(result);
        step4(result);
        step5a(result);
        step5b(result);
        
        return result;
    }
};

int main(int argc, char* argv[]) {
    if (argc < 3) {
        std::cerr << "Usage: " << argv[0] << " <input_file> <output_file>" << std::endl;
        return 1;
    }

    std::ifstream input(argv[1]);
    std::ofstream output(argv[2]);
    
    if (!input.is_open()) {
        std::cerr << "Error opening input file: " << argv[1] << std::endl;
        return 1;
    }
    
    if (!output.is_open()) {
        std::cerr << "Error opening output file: " << argv[2] << std::endl;
        return 1;
    }

    PorterStemmer stemmer;
    std::string token;
    size_t count = 0;

    while (std::getline(input, token)) {
        if (!token.empty()) {
            std::string stemmed = stemmer.stem(token);
            output << stemmed << "\n";
            count++;
            if (count % 10000 == 0) {
                std::cerr << "Processed " << count << " tokens..." << std::endl;
            }
        }
    }

    input.close();
    output.close();
    
    std::cout << "Stemming completed. Processed " << count << " tokens." << std::endl;
    return 0;
}

