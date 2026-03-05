document.addEventListener('DOMContentLoaded', () => {
  const patternInput = document.getElementById('pattern');
  const generateBtn = document.getElementById('generateBtn');
  const clearBtn = document.getElementById('clearBtn');
  
  // ✅ Новый паттерн по умолчанию с lookbehind
  const defaultPattern = '(?<![уУэЭ]|[тТ][иИ])[пП][еЕиИ]?[зЗ3жЖ][дДтТ]';
  patternInput.value = defaultPattern;
  
  if (clearBtn) {
    clearBtn.addEventListener('click', () => {
      patternInput.value = '';
      patternInput.focus();
      document.getElementById('accepted').innerHTML = '';
      document.getElementById('rejected').innerHTML = '';
    });
  }
  
  generateBtn.addEventListener('click', generateWords);
  patternInput.addEventListener('keypress', e => {
    if (e.key === 'Enter') generateWords();
  });
});

function generateWords() {
  const pattern = document.getElementById('pattern').value.trim();
  if (!pattern) return;
  
  document.getElementById('accepted').innerHTML = '⏳';
  document.getElementById('rejected').innerHTML = '';
  document.getElementById('generateBtn').disabled = true;
  
  fetch(`/generate?pattern=${encodeURIComponent(pattern)}`)
    .then(r => r.json())
    .then(data => {
      const accepted = (data.accepted || []).map(w => 
        `<span class="word">${w}</span>`
      ).join(' ');
      
      const rejected = (data.rejected || []).map(w => 
        `<span class="word">${w}</span>`
      ).join(' ');
      
      document.getElementById('accepted').innerHTML = accepted;
      document.getElementById('rejected').innerHTML = rejected;
    })
    .catch(() => {
      document.getElementById('accepted').innerHTML = 'Ошибка';
      document.getElementById('rejected').innerHTML = '';
    })
    .finally(() => {
      document.getElementById('generateBtn').disabled = false;
    });
}