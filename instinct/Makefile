CXX      := g++
CXXFLAGS := -std=c++17 -O3 -march=native -Wall -Wextra
TARGET   := instinct_classifier

.PHONY: all clean weights

all: $(TARGET)

# generate model_weights.h from chaos1 pkl files
# run: make weights CHAOS1=/path/to/chaos1
weights:
	cd $(CHAOS1) && python3 $(CURDIR)/export_weights.py

$(TARGET): classifier.cpp model_weights.h
	$(CXX) $(CXXFLAGS) -o $@ $<
	@echo "[instinct] built: $(TARGET)"

clean:
	rm -f $(TARGET) model_weights.h
