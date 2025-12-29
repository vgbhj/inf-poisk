import sys
import pandas as pd
import matplotlib.pyplot as plt
import numpy as np

if len(sys.argv) < 2:
    print("Usage: python3 plot_zipf.py <frequency_file> [output_image]")
    sys.exit(1)

input_file = sys.argv[1]
output_file = sys.argv[2] if len(sys.argv) > 2 else "zipf_plot.png"

try:
    df = pd.read_csv(input_file, sep='\t', encoding='utf-8', encoding_errors='replace')
except Exception as e:
    print(f"Error: {e}")
    sys.exit(1)

df = df.dropna(subset=['rank', 'frequency'])
ranks = df['rank'].values
frequencies = df['frequency'].values

c_zipf = frequencies[0]
zipf_theoretical = c_zipf / ranks

plt.figure(figsize=(10, 6))

plt.loglog(ranks, frequencies, 'b-', label='Actual Data (Log-Log)', linewidth=1.5)
plt.loglog(ranks, zipf_theoretical, 'r--', label="Zipf's Law (Theoretical)", linewidth=1.5)

plt.xlabel('Rank (log scale)')
plt.ylabel('Frequency (log scale)')
plt.title("Zipf's Law Distribution (Log-Log Scale)")
plt.legend()
plt.grid(True, which="both", ls="-", alpha=0.2)

plt.tight_layout()
plt.savefig(output_file, dpi=300)
print(f"Saved to {output_file}")

print("\n--- Анализ закона Ципфа ---")
print(f"Наиболее частотное слово: '{df.iloc[0]['term']}' с частотой {frequencies[0]}")
print(f"Ожидаемая частота 2-го слова: {zipf_theoretical[1]:.2f}, фактическая: {frequencies[1]}")