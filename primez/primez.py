# The number of primes in [1..N] is approx. equal to N / log(N).

import math

def sieve(n):
    """Returns an array A of booleans in which A[i] == True iff i is prime. len(A) is n + 1."""
    s = [True] * (n + 1)
    s[0] = s[1] = False  # 0 and 1 are not prime.
    m = int(math.sqrt(n))
    for i in range(4, len(s), 2):
        # No even number is prime except 2
        s[i] = False
    for i in range(3, m + 1, 2):
        if not s[i]:
            continue  # already processed via a prime factor of i.
        for j in range(i + i, len(s), i):
            s[j] = False
    return s

def exact_pi(n, s=None):
    if s is None:
        s = sieve(n)
    return sum(s[:n+1])


def approx_pi(n):
    return n / math.log(n)
        

def approx_pi_err(n):
    s = sieve(n)
    p = approx_pi(n)
    a = exact_pi(n, s)
    return (p - a) / a


def main():
    for i in range(1, 9):
        n = 10**i
        err = approx_pi_err(n)
        print(f"Error at {n}: {err * 100:.3f}%")

if __name__ == "__main__":
    main()