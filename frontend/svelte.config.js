import preprocess from 'svelte-preprocess';

const config = {
  // Consult https://github.com/sveltejs/svelte-preprocess
  // for more information about preprocessors
  preprocess: preprocess({
    // Enable processing of <script lang="ts"> blocks
    typescript: true,
    // Enable processing of <style lang="scss"> or other CSS preprocessors if needed
    // scss: true,
    // Enable PostCSS processing if needed
    // postcss: true,
  })
};

export default config;
