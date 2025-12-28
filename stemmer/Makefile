CXX = g++
CXXFLAGS = -std=c++17 -O2 -Wall
TARGET = stemmer
SOURCES = stemmer.cpp

$(TARGET): $(SOURCES)
	$(CXX) $(CXXFLAGS) -o $(TARGET) $(SOURCES)

clean:
	rm -f $(TARGET)

.PHONY: clean

