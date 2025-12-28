#include <iostream>
#include <fstream>
#include <string>
#include <map>
#include <vector>
#include <algorithm>
#include <cmath>

struct TermFreq {
    std::string term;
    size_t frequency;
    
    bool operator<(const TermFreq& other) const {
        return frequency > other.frequency;
    }
};

int main(int argc, char* argv[]) {
    if (argc < 3) {
        std::cerr << "Usage: " << argv[0] << " <input_file> <output_file>" << std::endl;
        return 1;
    }

    std::ifstream input(argv[1]);
    if (!input.is_open()) {
        std::cerr << "Error opening input file: " << argv[1] << std::endl;
        return 1;
    }

    std::map<std::string, size_t> frequencies;
    std::string token;
    size_t totalTokens = 0;

    std::cerr << "Counting frequencies..." << std::endl;
    while (std::getline(input, token)) {
        if (!token.empty()) {
            frequencies[token]++;
            totalTokens++;
            if (totalTokens % 100000 == 0) {
                std::cerr << "Processed " << totalTokens << " tokens..." << std::endl;
            }
        }
    }
    input.close();

    std::vector<TermFreq> sorted;
    for (const auto& pair : frequencies) {
        sorted.push_back({pair.first, pair.second});
    }
    std::sort(sorted.begin(), sorted.end());

    std::ofstream output(argv[2]);
    if (!output.is_open()) {
        std::cerr << "Error opening output file: " << argv[2] << std::endl;
        return 1;
    }

    output << "rank\tterm\tfrequency\tlog_rank\tlog_frequency" << std::endl;
    
    for (size_t i = 0; i < sorted.size(); ++i) {
        size_t rank = i + 1;
        double logRank = std::log10(static_cast<double>(rank));
        double logFreq = std::log10(static_cast<double>(sorted[i].frequency));
        
        output << rank << "\t" << sorted[i].term << "\t" << sorted[i].frequency 
               << "\t" << logRank << "\t" << logFreq << std::endl;
    }
    
    output.close();

    std::cout << "Total unique terms: " << sorted.size() << std::endl;
    std::cout << "Total tokens: " << totalTokens << std::endl;
    std::cout << "Results saved to: " << argv[2] << std::endl;

    return 0;
}

