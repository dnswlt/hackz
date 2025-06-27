from collections import Counter
import os
import sys

def main():
    c = Counter()
    for dirpath, dirnames, filenames in os.walk(sys.argv[1]):
        for f in filenames:
            _, ext = os.path.splitext(f)
            c[ext] += 1
    print(c)

if __name__ == '__main__':
    main()