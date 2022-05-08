"""Figure out how many cores and/or CPUs you have by running 1..N CPU-bound
processes and measuring thoughput.
"""

from multiprocessing import Pool
import time

def fib(n):
    if n <= 1: return 1
    return fib(n-1) + fib(n-2)

def main():
    for i in range(1, 15):
        with Pool(i) as p:
            before = time.perf_counter()
            p.map(fib, [35] * i)
            duration = time.perf_counter() - before
            print(f"At {i} processes: {duration:.3f}")

if __name__ == "__main__":
    main()