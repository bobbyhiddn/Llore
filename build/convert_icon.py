from PIL import Image
import os

# Ensure the windows directory exists
os.makedirs('windows', exist_ok=True)

# Open the PNG image
img = Image.open('../frontend/src/assets/images/logo.png')

# Convert to RGBA if not already
img = img.convert('RGBA')

# Resize the image to be 1.75x larger for the base size
base_size = int(256 * 1.75)
img = img.resize((base_size, base_size), Image.Resampling.LANCZOS)

# Create ICO file with multiple sizes, starting from the larger base
img.save('windows/icon.ico', format='ICO', sizes=[(base_size, base_size), (256,256), (128,128), (64,64), (48,48), (32,32)])
