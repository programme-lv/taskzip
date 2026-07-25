#include "testlib.h"
#include <bits/stdc++.h>

using namespace std;

int main(int argc, char *argv[]) {
  registerValidation(argc, argv);
  int N = inf.readInt(2, 5, "N");
  inf.readEoln();

  int directionCount = 0;
  int berryCount = 0;

  for (int i = 0; i < N; i++) {
    for (int j = 0; j < N; j++) {
      char c = inf.readChar();
      inf.ensuref(c == '.' || c == '#' || c == '*' || c == '>' || c == '<' ||
                      c == '^' || c == 'v',
                  "Invalid character '%c' at position (%d, %d)", c, i + 1,
                  j + 1);

      // Count direction symbols (snake head)
      if (c == '>' || c == '<' || c == '^' || c == 'v') {
        directionCount++;
      }

      // Count berries
      if (c == '*') {
        berryCount++;
      }
    }
    inf.readEoln();
  }

  // Validate exactly 1 direction symbol
  inf.ensuref(directionCount == 1,
              "Expected exactly 1 direction symbol (>,<,^,v), found %d",
              directionCount);

  // Validate berry count: 1 <= d <= 8
  inf.ensuref(berryCount >= 1 && berryCount <= 8,
              "Berry count must be between 1 and 8, found %d", berryCount);

  // Subtask-specific constraints
  if (validator.group() == "1") {
    ensuref(N == 2, "Subtask 2: N must be 2");
  } else if (validator.group() == "2") {
    ensuref(N == 3, "Subtask 3: N must be 3");
  } else if (validator.group() == "3") {
    ensuref(berryCount == 1, "Subtask 4: must have exactly 1 berry");
  } else if (validator.group() == "4") {
    ensuref(N == 4, "Subtask 5: N must be 4");
  }

  inf.readEof();
  return 0;
}
