success_count = 0
total_count = 0

with open('jdk-25-build-results.csv', encoding='utf-8') as f:
    next(f)  # skip header
    for line in f:
        parts = line.strip().split(',')
        if len(parts) == 3:
            total_count += 1
            if parts[2] == 'success':
                success_count += 1

success_pct = (success_count / total_count) * 100
not_success_pct = 100 - success_pct

print(f"Successful builds: {success_count}")
print(f"Total builds: {total_count}")
print(f"Success percentage: {success_pct:.2f}%")
print(f"Not successful percentage: {not_success_pct:.2f}%")
