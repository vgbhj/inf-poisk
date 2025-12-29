#ifndef STRING_H
#define STRING_H

#include <cstring>
#include <cstddef>

class String {
private:
    char* data;
    size_t len;

public:
    String() : data(nullptr), len(0) {}

    String(const char* str) {
        if (str) {
            len = strlen(str);
            data = new char[len + 1];
            strcpy(data, str);
        } else {
            data = nullptr;
            len = 0;
        }
    }

    String(const String& other) {
        if (other.data) {
            len = other.len;
            data = new char[len + 1];
            strcpy(data, other.data);
        } else {
            data = nullptr;
            len = 0;
        }
    }

    ~String() {
        if (data) {
            delete[] data;
            data = nullptr;
        }
    }

    String& operator=(const String& other) {
        if (this != &other) {
            if (data) {
                delete[] data;
                data = nullptr;
            }
            if (other.data) {
                len = other.len;
                data = new char[len + 1];
                strcpy(data, other.data);
            } else {
                data = nullptr;
                len = 0;
            }
        }
        return *this;
    }

    bool operator==(const String& other) const {
        if (len != other.len) return false;
        if (data == nullptr && other.data == nullptr) return true;
        if (data == nullptr || other.data == nullptr) return false;
        return strcmp(data, other.data) == 0;
    }

    bool operator==(const char* str) const {
        if (data == nullptr && str == nullptr) return true;
        if (data == nullptr || str == nullptr) return false;
        return strcmp(data, str) == 0;
    }

    bool operator<(const String& other) const {
        if (data == nullptr && other.data == nullptr) return false;
        if (data == nullptr) return true;
        if (other.data == nullptr) return false;
        return strcmp(data, other.data) < 0;
    }

    const char* c_str() const {
        return data ? data : "";
    }

    size_t length() const {
        return len;
    }

    bool empty() const {
        return len == 0;
    }
};

#endif
