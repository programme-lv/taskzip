#include <cstdlib>
#include <ctime>
#include <iostream>
#include <string>
#include <sys/mman.h>
#include <unistd.h>

int special(const std::string &mode) {
    if (mode == "close-input") close(STDIN_FILENO);
    if (mode == "eof") return 0;
    if (mode == "crash" || mode == "accept-crash") std::abort();
    if (mode == "hang") while (true) pause();
    if (mode == "spin") while (true) {}
    if (mode == "memory") {
        void *p = mmap(nullptr, 512 * 1024 * 1024, PROT_READ | PROT_WRITE,
                       MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
        if (p == MAP_FAILED) return 17;
        munmap(p, 512 * 1024 * 1024);
    }
    if (mode == "cpu") {
        auto end = std::clock() + CLOCKS_PER_SEC / 5;
        while (std::clock() < end) {}
    }
    return -1;
}

int main() {
    std::string mode;
    int n;
    std::cin >> mode >> n;
    int status = special(mode);
    if (status != -1) return status;
    if (mode == "stderr") std::cerr << std::string(200000, 'x') << '\n';
    if (mode == "wrong") ++n;
    std::cout << n + 1 << std::endl;
    if (mode != "judge-hang" && mode != "close-output" && mode != "close-input") {
        std::string reply;
        if (!(std::cin >> reply) || reply != "done") return 9;
    }
}
