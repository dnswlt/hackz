from itertools import permutations

RULES = [
    ('n', 'r', 'f', 'e'),
    ('f', 'i', 'r', 'n'),
    ('u', 'f', 'e', 'i'),
    ('f', 'n', 'r', 'i'),
    ('r', 'n', 'f', 'e'),
    ('u', 'n', 'i', 'e'),
    ('n', 'e', 'r', 'f'),
    ('r', 'u', 'e', 'f'),
]

def sat(order, rule):
    p1 = order[rule[0]]
    p2 = order[rule[1]]
    p3 = order[rule[2]]
    p4 = order[rule[3]]
    return p1 > p2 or p3 < p4

def mkorders():
    ps = sorted(set([p for r in RULES for p in r]))
    perms = list(permutations(ps))
    orders = []
    for perm in perms:
        d = {}
        for i, p in enumerate(perm):
            d[p] = i
        orders.append(d)
    return orders

orders = mkorders()
for order in orders:
    if all(sat(order, r) for r in RULES):
        print(order)