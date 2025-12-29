#ifndef VECTOR_H
#define VECTOR_H

#include <cstring>
#include <cstddef>

template<typename T>
class Vector {
private:
    T* data;
    size_t size;
    size_t capacity;

    void resize() {
        size_t new_capacity = capacity == 0 ? 16 : capacity * 2;
        T* newData = new T[new_capacity];
        for (size_t i = 0; i < size; ++i) {
            newData[i] = data[i];
        }
        if (data) delete[] data;
        data = newData;
        capacity = new_capacity;
    }

public:
    Vector() : data(nullptr), size(0), capacity(0) {}

    ~Vector() {
        if (data) {
            delete[] data;
            data = nullptr;
        }
    }

    Vector(const Vector& other) {
        size = other.size;
        capacity = other.capacity;
        if (capacity > 0) {
            data = new T[capacity];
            for (size_t i = 0; i < size; ++i) {
                data[i] = other.data[i];
            }
        } else {
            data = nullptr;
        }
    }

    Vector& operator=(const Vector& other) {
        if (this != &other) {
            if (data) delete[] data;
            size = other.size;
            capacity = other.capacity;
            if (capacity > 0) {
                data = new T[capacity];
                for (size_t i = 0; i < size; ++i) {
                    data[i] = other.data[i];
                }
            } else {
                data = nullptr;
            }
        }
        return *this;
    }

    void push_back(const T& value) {
        if (size >= capacity) {
            resize();
        }
        data[size++] = value;
    }

    T& operator[](size_t index) {
        return data[index];
    }

    const T& operator[](size_t index) const {
        return data[index];
    }

    size_t get_size() const {
        return size;
    }

    bool empty() const {
        return size == 0;
    }

    void clear() {
        size = 0;
    }

    T* begin() {
        return data;
    }

    T* end() {
        return data + size;
    }

    const T* begin() const {
        return data;
    }

    const T* end() const {
        return data + size;
    }
};

#endif
