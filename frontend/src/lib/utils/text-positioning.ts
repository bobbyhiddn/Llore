// Cached mirror div to avoid recreating it on every mouse move
let mirrorDiv: HTMLDivElement | null = null;
const mirrorId = 'llore-text-mirror';

// Cache for the last getWordAtPoint call
let lastWordCache: {
  clientX: number;
  clientY: number;
  result: WordInfo | null;
} | null = null;

// List of CSS properties to copy from the textarea to the mirror div for accurate measurement
const relevantStyles: string[] = [
  'font-family', 'font-size', 'font-weight', 'font-style',
  'line-height', 'letter-spacing', 'word-spacing',
  'text-transform',
  'border-top-width', 'border-right-width', 'border-bottom-width', 'border-left-width',
  'box-sizing', 'white-space', 'overflow-wrap', 'word-break'
];

/**
 * Creates or retrieves a hidden "mirror" div that mimics the styles of the target element.
 */
function getMirror(element: HTMLTextAreaElement): HTMLDivElement {
  mirrorDiv = document.getElementById(mirrorId) as HTMLDivElement | null;
  if (!mirrorDiv) {
    mirrorDiv = document.createElement('div');
    mirrorDiv.id = mirrorId;
    document.body.appendChild(mirrorDiv);
  }

  const style = window.getComputedStyle(element);
  mirrorDiv.style.position = 'absolute';
  mirrorDiv.style.left = '-9999px'; // Position off-screen
  mirrorDiv.style.top = '0px';
  
  // Calculate the exact content width (clientWidth minus padding)
  const paddingLeft = parseFloat(style.paddingLeft) || 0;
  const paddingRight = parseFloat(style.paddingRight) || 0;
  const contentWidth = element.clientWidth - paddingLeft - paddingRight;
  
  mirrorDiv.style.width = `${contentWidth}px`; // Match exact content width
  mirrorDiv.style.height = 'auto';
  mirrorDiv.style.visibility = 'hidden';
  mirrorDiv.style.whiteSpace = 'pre-wrap';
  mirrorDiv.style.wordWrap = 'break-word';
  
  // Copy all relevant styles to ensure exact matching
  relevantStyles.forEach(prop => {
    mirrorDiv!.style.setProperty(prop, style.getPropertyValue(prop));
  });

  // Ensure the mirror div has the same text alignment as the textarea (always left for textarea)
  mirrorDiv.style.textAlign = 'left';
  
  // Reset any inherited styles that might affect positioning
  mirrorDiv.style.margin = '0';
  mirrorDiv.style.display = 'block';
  mirrorDiv.style.textIndent = '0';
  mirrorDiv.style.direction = 'ltr';
  mirrorDiv.style.padding = '0'; // Reset padding since we handle it separately

  return mirrorDiv;
}

/**
 * Gets the character index within an element at a given screen coordinate.
 * Uses the modern caretRangeFromPoint API for accuracy.
 * @returns The character index or -1 if not found.
 */
export function getCharIndexAtPoint(element: HTMLTextAreaElement, clientX: number, clientY: number): number {
  // First try the modern API approach
  if (document.caretRangeFromPoint) {
    const range = document.caretRangeFromPoint(clientX, clientY);
    if (range) {
      // For textarea elements, we need to calculate the text position differently
      const rect = element.getBoundingClientRect();
      const relativeX = clientX - rect.left;
      const relativeY = clientY - rect.top;
      
      // Use the existing getCursorPositionFromMouseEvent logic but adapted
      return getCursorPositionFromCoords(element, relativeX, relativeY);
    }
  }
  
  // Fallback to coordinate-based calculation
  const rect = element.getBoundingClientRect();
  const relativeX = clientX - rect.left;
  const relativeY = clientY - rect.top;
  
  return getCursorPositionFromCoords(element, relativeX, relativeY);
}

/**
 * Helper function to get cursor position from relative coordinates
 * (adapted from the existing WriteView logic)
 */
function getCursorPositionFromCoords(textarea: HTMLTextAreaElement, x: number, y: number): number {
  const text = textarea.value;
  const style = window.getComputedStyle(textarea);

  // Get accurate coordinates and styles
  const adjustedX = x - parseFloat(style.paddingLeft);
  const adjustedY = y - parseFloat(style.paddingTop) + textarea.scrollTop;

  // Create a hidden div to mirror textarea styles
  const mirrorDiv = getMirror(textarea);
  
  let position = -1;
  // Use a sentinel character to ensure we can always find a range
  mirrorDiv.textContent = text + '|';
  const range = document.createRange();
  const textNode = mirrorDiv.childNodes[0];
  
  if (!textNode || !textNode.textContent) {
    return text.length; // Fallback if textNode is not found
  }

  // Iterate through characters to find the one at the click coordinates
  for (let i = 0; i < textNode.textContent.length; i++) {
    range.setStart(textNode, i);
    range.setEnd(textNode, i + 1);
    const rangeRect = range.getBoundingClientRect();
    const mirrorRect = mirrorDiv.getBoundingClientRect();

    // Calculate position relative to the mirror div
    const relativeX = rangeRect.left - mirrorRect.left;
    const relativeY = rangeRect.top - mirrorRect.top;

    // Check if the click is within the vertical bounds of the current character's line
    if (adjustedY >= relativeY && adjustedY <= relativeY + rangeRect.height) {
      // Check if the click is to the left of the character's midpoint
      if (adjustedX < relativeX + rangeRect.width / 2) {
        position = i;
        break;
      }
    } else if (adjustedY < relativeY) {
      // Click is on a previous line, so we've gone too far
      position = i > 0 ? i : 0;
      break;
    }
  }
  
  if (position === -1) {
    position = text.length; // Clicked past the last character
  }

  return position > text.length ? text.length : position;
}

export interface WordInfo {
  word: string;
  rect: DOMRect;
  index: number; // Starting character index of the word
}

/**
 * Finds the full word, its starting index, and its bounding box at a given coordinate.
 */
export function getWordAtPoint(element: HTMLTextAreaElement, clientX: number, clientY: number): WordInfo | null {
  // Check cache first
  if (lastWordCache && 
      Math.abs(lastWordCache.clientX - clientX) < 5 && 
      Math.abs(lastWordCache.clientY - clientY) < 5) {
    return lastWordCache.result;
  }

  const text = element.value;
  const pos = getCharIndexAtPoint(element, clientX, clientY);
  
  if (pos === -1 || pos >= text.length) {
    lastWordCache = { clientX, clientY, result: null };
    return null;
  }

  // Check if we're over whitespace
  if (/\s/.test(text[pos])) {
    lastWordCache = { clientX, clientY, result: null };
    return null;
  }

  // Find the boundaries of the word at the determined character position
  let start = pos;
  while (start > 0 && /\S/.test(text[start - 1])) {
    start--;
  }
  let end = pos;
  while (end < text.length && /\S/.test(text[end])) {
    end++;
  }
  
  // Now we have the word's start/end index. Use the mirror to get its rect.
  const mirror = getMirror(element);
  const wordText = text.substring(start, end);

  // Recreate content with a span around the target word for measurement
  const sanitize = (str: string) => str.replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\n/g, '<br />');
  mirror.innerHTML = `${sanitize(text.substring(0, start))}<span id="target-word">${sanitize(wordText)}</span>${sanitize(text.substring(end))}`;

  const targetSpan = mirror.querySelector<HTMLSpanElement>('#target-word');
  if (!targetSpan) return null;

  // Calculate the final rect relative to the viewport
  const elementRect = element.getBoundingClientRect();
  const style = window.getComputedStyle(element);
  const paddingLeft = parseFloat(style.paddingLeft);
  const paddingTop = parseFloat(style.paddingTop);
  
  // Calculate position more directly using offsetLeft/offsetTop
  const finalRect = new DOMRect(
    elementRect.left + paddingLeft + targetSpan.offsetLeft - element.scrollLeft,
    elementRect.top + paddingTop + targetSpan.offsetTop - element.scrollTop,
    targetSpan.offsetWidth,
    targetSpan.offsetHeight
  );

  const result = {
    word: wordText,
    rect: finalRect,
    index: start,
  };

  // Cache the result
  lastWordCache = {
    clientX,
    clientY,
    result
  };

  return result;
}