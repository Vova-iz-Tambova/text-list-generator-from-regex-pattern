const STORAGE_KEYS = {
  PATTERNS_TEXT: 'regex-generator-patterns-text',
  FILTERS: 'regex-generator-filters'
};

let abortController = null;

document.addEventListener('DOMContentLoaded', () => {
  try {
    const patternsText = document.getElementById('patternsText');
    const generateBtn = document.getElementById('generateBtn');
    const clearBtn = document.getElementById('clearBtn');
    const pasteBtn = document.getElementById('pasteBtn');
    const copyBtn = document.getElementById('copyBtn');
    const loadBtn = document.getElementById('loadBtn');
    const fileInput = document.getElementById('fileInput');

    if (!patternsText) throw new Error('patternsText not found');
    if (!generateBtn) throw new Error('generateBtn not found');
    if (!clearBtn) throw new Error('clearBtn not found');
    if (!pasteBtn) throw new Error('pasteBtn not found');
    if (!copyBtn) throw new Error('copyBtn not found');
    if (!loadBtn) throw new Error('loadBtn not found');
    if (!fileInput) throw new Error('fileInput not found');

    // Load saved patterns or leave empty
    const savedText = localStorage.getItem(STORAGE_KEYS.PATTERNS_TEXT);
    if (savedText && savedText.trim().length > 0) {
      patternsText.value = savedText;
    } else {
      patternsText.value = '';
    }

    loadFilters();

    patternsText.addEventListener('input', () => {
      savePatternsText();
      autoResizeTextarea(patternsText);
    });

    loadBtn.addEventListener('click', () => {
      fileInput.click();
    });

    fileInput.addEventListener('change', (e) => {
      const file = e.target.files[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = (event) => {
        const content = event.target.result;
        const patterns = extractPatternsFromText(content);
        if (patterns.length > 0) {
          patternsText.value = patterns.join('\n');
          savePatternsText();
          autoResizeTextarea(patternsText);
        } else {
          alert('В файле не найдены паттерны. Ожидается формат: const PATTERN_NAME = \'...\' или каждая строка — отдельный паттерн.');
        }
        fileInput.value = '';
      };
      reader.readAsText(file);
    });

    pasteBtn.addEventListener('click', async () => {
      try {
        const text = await navigator.clipboard.readText();
        patternsText.value = text;
        savePatternsText();
        autoResizeTextarea(patternsText);
      } catch (e) {
        alert('Не удалось вставить. Разрешите доступ к буферу обмена.');
      }
    });

    copyBtn.addEventListener('click', async () => {
      try {
        await navigator.clipboard.writeText(patternsText.value);
      } catch (e) {
        alert('Не удалось скопировать. Выделите текст и нажмите Ctrl+C.');
      }
    });

    clearBtn.addEventListener('click', () => {
      patternsText.value = '';
      localStorage.removeItem(STORAGE_KEYS.PATTERNS_TEXT);
      document.getElementById('accepted').innerHTML = '';
      document.getElementById('rejected').innerHTML = '';
      autoResizeTextarea(patternsText);
    });

    generateBtn.addEventListener('click', () => {
      if (abortController) {
        abortController.abort();
        abortController = null;
        resetGenerateButton();
        return;
      }
      generateWords();
    });

    autoResizeTextarea(patternsText);

    ['exclude_uppercase', 'exclude_latin', 'exclude_digits', 'exclude_special'].forEach(id => {
      document.getElementById(id).addEventListener('change', saveFilters);
    });

  } catch (error) {
    console.error('[Init] Error during initialization:', error);
    alert('Ошибка загрузки интерфейса: ' + error.message);
  }
});

function autoResizeTextarea(textarea) {
  textarea.style.height = 'auto';
  textarea.style.height = textarea.scrollHeight + 'px';
}

function savePatternsText() {
  const text = document.getElementById('patternsText').value;
  localStorage.setItem(STORAGE_KEYS.PATTERNS_TEXT, text);
}

function loadFilters() {
  const saved = localStorage.getItem(STORAGE_KEYS.FILTERS);
  if (saved) {
    try {
      const filters = JSON.parse(saved);
      document.getElementById('exclude_uppercase').checked = filters.exclude_uppercase ?? true;
      document.getElementById('exclude_latin').checked = filters.exclude_latin ?? true;
      document.getElementById('exclude_digits').checked = filters.exclude_digits ?? true;
      document.getElementById('exclude_special').checked = filters.exclude_special ?? true;
    } catch (e) {}
  }
}

function saveFilters() {
  const filters = {
    exclude_uppercase: document.getElementById('exclude_uppercase').checked,
    exclude_latin: document.getElementById('exclude_latin').checked,
    exclude_digits: document.getElementById('exclude_digits').checked,
    exclude_special: document.getElementById('exclude_special').checked
  };
  localStorage.setItem(STORAGE_KEYS.FILTERS, JSON.stringify(filters));
}

// Extract pattern from a single line
// Returns pattern string if line has format: const PATTERN = '...' or const PATTERN = "..."
// Returns null otherwise
function extractPatternFromLine(line) {
  const trimmed = line.trim();
  if (trimmed.length === 0) return null;
  if (trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*')) return null;

  // Only process lines that have const PATTERN = '...' format
  const constMatch = trimmed.match(/^const\s+\w+\s*=\s*(['"])/);
  if (!constMatch) return null; // Not a const pattern line, skip

  const quoteChar = constMatch[1];
  const afterEquals = trimmed.indexOf('=', constMatch.index) + 1;
  const quotePos = trimmed.indexOf(quoteChar, afterEquals);
  
  if (quotePos === -1) return null; // No opening quote found

  // Find closing quote (same character)
  const lastQuote = trimmed.lastIndexOf(quoteChar);
  if (lastQuote <= quotePos) {
    // No closing quote on same line, take everything after opening quote
    const pattern = trimmed.substring(quotePos + 1).trim();
    return pattern || null;
  }

  // Extract pattern between quotes
  const pattern = trimmed.substring(quotePos + 1, lastQuote).trim();
  return pattern || null;
}

// Extract patterns from file content
// Only extracts lines with format: const PATTERN = '...' or const PATTERN = "..."
function extractPatternsFromText(text) {
  const patterns = [];
  const lines = text.split('\n');

  for (const line of lines) {
    const pattern = extractPatternFromLine(line);
    if (pattern) {
      patterns.push(pattern);
    }
  }

  return patterns;
}

// Check if string looks like a regex pattern
function isLikelyRegex(str) {
  // Must contain at least one regex special character
  return /[\[\](){}*+?.\\^$|?]/.test(str);
}

// Get patterns from textarea (each non-empty line is a pattern)
function getPatterns() {
  const text = document.getElementById('patternsText').value;
  const patterns = [];

  text.split('\n').forEach(line => {
    const trimmed = line.trim();
    if (trimmed.length === 0) return;
    if (trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*')) return;

    // Assume each line is a pattern
    patterns.push(trimmed);
  });

  return patterns;
}

function setGenerateButtonCancelling() {
  const btn = document.getElementById('generateBtn');
  btn.textContent = 'Отменить';
  btn.classList.add('cancel');
  btn.disabled = false;
}

function resetGenerateButton() {
  const btn = document.getElementById('generateBtn');
  btn.textContent = 'Сгенерировать';
  btn.classList.remove('cancel');
  btn.disabled = false;
}

function generateWords() {
  const patterns = getPatterns();

  if (patterns.length === 0) {
    alert('Введите хотя бы один паттерн');
    return;
  }

  document.getElementById('accepted').innerHTML = '⏳ Генерация...';
  document.getElementById('rejected').innerHTML = '';

  setGenerateButtonCancelling();

  abortController = new AbortController();
  const signal = abortController.signal;

  const params = new URLSearchParams();
  patterns.forEach(p => params.append('patterns', p));
  params.append('exclude_uppercase', document.getElementById('exclude_uppercase').checked);
  params.append('exclude_digits', document.getElementById('exclude_digits').checked);
  params.append('exclude_special', document.getElementById('exclude_special').checked);
  params.append('exclude_latin', document.getElementById('exclude_latin').checked);

  fetch(`/generate?${params}`, { signal })
    .then(r => {
      if (!r.ok) {
        throw new Error(`HTTP error! status: ${r.status}`);
      }
      return r.json();
    })
    .then(data => {
      if (data.results) {
        processMultiResults(data.results);
      } else {
        displayResults(data.accepted || [], data.rejected || []);
      }
    })
    .catch(err => {
      if (err.name === 'AbortError') {
        document.getElementById('accepted').innerHTML = '⚠️ Генерация отменена';
      } else {
        document.getElementById('accepted').innerHTML = 'Ошибка соединения: ' + err.message;
        document.getElementById('rejected').innerHTML = '';
      }
    })
    .finally(() => {
      abortController = null;
      resetGenerateButton();
    });
}

function processMultiResults(results) {
  const acceptedSet = new Set();
  const rejectedSet = new Set();
  const errors = [];

  if (!Array.isArray(results)) {
    return;
  }

  results.forEach((result, index) => {
    if (result.error) {
      errors.push(`Паттерн ${index + 1}: ${result.error}`);
    } else {
      (result.accepted || []).forEach(w => acceptedSet.add(w));
      (result.rejected || []).forEach(w => rejectedSet.add(w));
    }
  });

  acceptedSet.forEach(w => rejectedSet.delete(w));

  if (errors.length > 0) {
    alert('Ошибки в паттернах:\n' + errors.join('\n'));
  }

  displayResults([...acceptedSet], [...rejectedSet]);
}

function displayResults(accepted, rejected) {
  const acceptedHtml = accepted.map(w =>
    `<span class="word">${w}</span>`
  ).join(' ');

  const rejectedHtml = rejected.map(w =>
    `<span class="word">${w}</span>`
  ).join(' ');

  document.getElementById('accepted').innerHTML = acceptedHtml || '(нет принятых слов)';
  document.getElementById('rejected').innerHTML = rejectedHtml || '(нет отклонённых слов)';
}
