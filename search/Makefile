CXX = g++
CXXFLAGS = -std=c++17 -O2 -Wall
TARGETS = index_builder searcher index_stats

all: $(TARGETS)

index_builder: index_builder.cpp boolean_index.h hashmap.h vector.h string.h posting_list.h
	$(CXX) $(CXXFLAGS) -o index_builder index_builder.cpp

searcher: searcher.cpp boolean_searcher.h boolean_index.h query_parser.h hashmap.h vector.h string.h posting_list.h
	$(CXX) $(CXXFLAGS) -o searcher searcher.cpp

index_stats: index_stats.cpp boolean_index.h hashmap.h vector.h string.h posting_list.h
	$(CXX) $(CXXFLAGS) -o index_stats index_stats.cpp

clean:
	rm -f $(TARGETS) *.o

.PHONY: all clean
