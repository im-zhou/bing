document.addEventListener('DOMContentLoaded', () => {
  const cookie = document.cookie.split('; ').find(v => v.startsWith('likes='));
  const liked = new Set(cookie ? decodeURIComponent(cookie.slice(6)).split(/[_,]/) : []);
  document.querySelectorAll('.ctrl.heart').forEach(button => {
    const id = button.getAttribute('photo');
    if (liked.has(id)) button.classList.add('active');
    const like = async () => {
      if (button.classList.contains('active') || button.dataset.busy) return;
      button.dataset.busy = '1';
      try {
        const response = await fetch(`/photo/${encodeURIComponent(id)}?force=like`, {headers: {'X-Requested-With': 'XMLHttpRequest'}});
        if (!response.ok) throw new Error('点赞失败，请稍后重试');
        const data = await response.json();
        if (data.code !== 200) throw new Error(data.msg || '点赞失败');
        button.classList.add('active');
        button.querySelector('em').textContent = Number(button.getAttribute('likes')) + 1;
      } catch (error) { window.alert(error.message); }
      finally { delete button.dataset.busy; }
    };
    button.addEventListener('click', like);
    button.addEventListener('keydown', event => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); like(); } });
  });
});
