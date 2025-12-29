#!/usr/bin/env python3
import sys
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

if len(sys.argv) < 2:
    print("Usage: python3 plot_zipf.py <frequency_file> [output_image]")
    sys.exit(1)

input_file = sys.argv[1]
output_file = sys.argv[2] if len(sys.argv) > 2 else "zipf_plot.png"

df = pd.read_csv(input_file, sep='\t')

ranks = df['rank'].values
frequencies = df['frequency'].values
log_ranks = df['log_rank'].values
log_frequencies = df['log_frequency'].values

max_freq = frequencies[0]
c_zipf = max_freq

zipf_frequencies = c_zipf / ranks
log_zipf_freq = np.log10(zipf_frequencies)

plt.figure(figsize=(12, 8))

plt.subplot(2, 1, 1)
plt.loglog(ranks, frequencies, 'b-', label='Actual data', linewidth=0.5, alpha=0.7)
plt.loglog(ranks, zipf_frequencies, 'r--', label=f'Zipf\'s law (C={c_zipf:.2f})', linewidth=2)
plt.xlabel('Rank (log scale)')
plt.ylabel('Frequency (log scale)')
plt.title('Zipf\'s Law: Frequency vs Rank')
plt.legend()
plt.grid(True, alpha=0.3)

plt.subplot(2, 1, 2)
plt.plot(log_ranks, log_frequencies, 'b-', label='Actual data', linewidth=0.5, alpha=0.7)
plt.plot(log_ranks, log_zipf_freq, 'r--', label=f'Zipf\'s law (C={c_zipf:.2f})', linewidth=2)
plt.xlabel('log(Rank)')
plt.ylabel('log(Frequency)')
plt.title('Zipf\'s Law: log-log plot')
plt.legend()
plt.grid(True, alpha=0.3)

plt.tight_layout()
plt.savefig(output_file, dpi=300, bbox_inches='tight')
print(f"Plot saved to {output_file}")

if len(sys.argv) > 3 and sys.argv[3] == "--mandelbrot":
    print("\nMandelbrot's law (optional):")
    print("f(r) = C / (r + beta)^alpha")
    print("To fit Mandelbrot, use scipy.optimize.curve_fit")
    print("Typical values: alpha ≈ 1, beta ≈ 2.7")

