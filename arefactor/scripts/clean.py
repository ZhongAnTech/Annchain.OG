import os

if __name__ == '__main__':
    for folder in os.listdir('../data'):
        for c in ['consensus_state.ledger', 'ledger.ledger', 'ledger_temp.ledger']:
            p = os.path.join('../data', folder, 'data', c)
            if os.path.exists(p):
                print(p)
                os.remove(p)