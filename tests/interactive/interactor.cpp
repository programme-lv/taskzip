#include "testlib.h"
#include <chrono>
#include <csignal>
#include <thread>

int main(int argc, char **argv) {
    registerInteraction(argc, argv);
    std::string mode = inf.readToken();
    int n = inf.readInt();
    if (mode == "judge-work") {
        auto end = std::chrono::steady_clock::now() + std::chrono::milliseconds(150);
        while (std::chrono::steady_clock::now() < end) {}
    }
    std::cout << mode << " " << n << std::endl;
    if (mode == "fail") quitf(_fail, "broken jury data");
    if (mode == "points") quitp(0.5, "partial");
    if (mode == "judge-signal") std::raise(SIGABRT);
    if (mode == "accept-crash") quitf(_ok, "accepted too early");
    int actual = ouf.readInt();
    if (mode == "judge-hang") while (true) std::this_thread::sleep_for(std::chrono::seconds(1));
    int expected = n + 1;
    if (!ans.seekEof()) expected = ans.readInt();
    tout << actual << std::endl;
    if (actual != expected) quitf(_wa, "expected %d, got %d", expected, actual);
    if (mode == "close-output") {
        if (!ouf.seekEof()) quitf(_wa, "trailing output");
        quitf(_ok, "EOF received");
    }
    std::cout << "done" << std::endl;
    quitf(_ok, "correct");
}
