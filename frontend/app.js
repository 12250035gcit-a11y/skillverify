/* ── SkillVerify app.js ─────────────────────────────────── */
const API_URL = window.location.origin;
let currentUser = null;
let authToken   = null;

/* ── Helpers ─────────────────────────────────────────────── */
const $      = id => document.getElementById(id);
const esc    = s  => String(s ?? '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
const cap    = s  => s ? s[0].toUpperCase() + s.slice(1) : '—';
const show   = (id, visible = true) => { const el=$(id); if(el) el.style.display = visible ? '' : 'none' };
const setTxt = (id, v) => { const el=$(id); if(el) el.textContent = v ?? '—'; };
const fmtDate = s => { try { return new Date(s).toLocaleDateString('en-US',{month:'short',day:'numeric',year:'numeric'}) } catch { return s } };

function showAlert(id, msg, type = 'err') {
  const el = $(id); if (!el) return;
  el.className = `alert alert-${type} show`;
  const spans = el.querySelectorAll('span');
  if (spans.length >= 2) spans[1].textContent = msg;
  else el.textContent = msg;
}
function hideAlert(id) { const el=$(id); if(el) el.classList.remove('show') }

function btnLoad(btn, on) {
  if (!btn) return;
  if (on) { btn.dataset.orig = btn.innerHTML; btn.innerHTML = '<span class="spin"></span> Please wait…'; btn.disabled = true; }
  else    { btn.innerHTML = btn.dataset.orig || btn.innerHTML; btn.disabled = false; }
}

/* eye / eye-off SVG paths */
const EYE_OPEN = `<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>`;
const EYE_OFF  = `<path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/>`;

function togglePassword(inputId, btn) {
  const input = $(inputId); if (!input) return;
  const showing = input.type === 'text';
  input.type = showing ? 'password' : 'text';
  const svg = btn.querySelector('svg');
  if (svg) svg.innerHTML = showing ? EYE_OPEN : EYE_OFF;
  btn.setAttribute('aria-label', showing ? 'Show password' : 'Hide password');
}

function showSuccessToast(msg) {
  const t = document.createElement('div');
  t.className = 'success-toast'; t.textContent = msg;
  document.body.appendChild(t);
  setTimeout(() => t.classList.add('show'), 50);
  setTimeout(() => { t.classList.remove('show'); setTimeout(() => t.remove(), 400); }, 3500);
}

/* ── Bootstrap ───────────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', () => {
  try {
    const tok = localStorage.getItem('token');
    const usr = localStorage.getItem('user');
    if (tok && usr) { authToken = tok; currentUser = JSON.parse(usr); }
  } catch { localStorage.removeItem('token'); localStorage.removeItem('user'); }

  initNav();

  const p = window.location.pathname;
  if ((p.includes('dashboard') || p.includes('test') || p.includes('admin')) && !currentUser) {
    window.location.href = 'login.html'; return;
  }

  if      (p.includes('admin'))     initAdminPage();
  else if (p.includes('dashboard')) loadDashboard();
  else if (p.includes('jobs'))      initJobsPage();
  else if (p.includes('test'))      setupTestPage();
  else if (p.includes('login'))     setupLoginForm();
  else if (p.includes('register'))  setupRegisterForm();
  else                              initHomePage();
});

/* ── Nav ─────────────────────────────────────────────────── */
function initNav() {
  const nav = document.querySelector('.nav');
  if (nav) window.addEventListener('scroll', () => nav.classList.toggle('pinned', scrollY > 8), { passive: true });

  const ham    = $('navHam');
  const drawer = $('navDrawer');
  const veil   = $('navVeil');
  const open  = () => { ham?.classList.add('open'); drawer?.classList.add('open'); veil?.classList.add('on'); document.body.style.overflow = 'hidden'; };
  const close = () => { ham?.classList.remove('open'); drawer?.classList.remove('open'); veil?.classList.remove('on'); document.body.style.overflow = ''; };

  ham?.addEventListener('click', () => drawer?.classList.contains('open') ? close() : open());
  veil?.addEventListener('click', close);
  document.querySelectorAll('.drawer-links a').forEach(a => a.addEventListener('click', close));

  if (currentUser) {
    show('navGuest', false); show('navUser');
    show('drawerGuest', false); show('drawerUser');

    // Inject Admin link for admin users (only on pages that don't already have it)
    if (currentUser.role === 'admin' && !window.location.pathname.includes('admin')) {
      const adminLi = `<li><a href="admin.html" style="color:var(--blue)">Admin</a></li>`;
      document.querySelectorAll('.nav-links, .drawer-links').forEach(ul => {
        ul.insertAdjacentHTML('beforeend', adminLi);
      });
    }
  }
}

function logout() {
  localStorage.removeItem('token'); localStorage.removeItem('user');
  window.location.href = 'index.html';
}

/* ── Login ───────────────────────────────────────────────── */
function setupLoginForm() {
  const form = $('loginForm'); if (!form) return;
  if (location.search.includes('registered=1')) showAlert('formAlert', 'Account created! Sign in below.', 'ok');

  form.addEventListener('submit', async e => {
    e.preventDefault(); hideAlert('formAlert');
    const btn = $('loginBtn'); btnLoad(btn, true);
    try {
      const r = await fetch(`${API_URL}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: $('email').value, password: $('password').value })
      });
      const d = await r.json();
      if (r.ok) {
        localStorage.setItem('token', d.token);
        localStorage.setItem('user', JSON.stringify(d.user));
        window.location.href = 'dashboard.html';
      } else {
        showAlert('formAlert', d.message || 'Login failed.');
        btnLoad(btn, false);
      }
    } catch { showAlert('formAlert', 'Network error — is the server running?'); btnLoad(btn, false); }
  });
}

/* ── Register ────────────────────────────────────────────── */
function setupRegisterForm() {
  const form = $('registerForm'); if (!form) return;

  document.querySelectorAll('.role-card').forEach(card => {
    card.addEventListener('click', () => {
      document.querySelectorAll('.role-card').forEach(c => c.classList.remove('on'));
      card.classList.add('on');
    });
  });

  form.addEventListener('submit', async e => {
    e.preventDefault(); hideAlert('formAlert');
    const btn = $('registerBtn'); btnLoad(btn, true);
    const role = document.querySelector('.role-card.on input[type=radio]')?.value || 'freelancer';
    try {
      const r = await fetch(`${API_URL}/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name:     $('fullName').value.trim(),
          phone:    $('phone').value.trim(),
          email:    $('email').value.trim(),
          password: $('password').value,
          role,
        })
      });
      const d = await r.json();
      if (r.ok) window.location.href = 'login.html?registered=1';
      else { showAlert('formAlert', d.message || 'Registration failed.'); btnLoad(btn, false); }
    } catch { showAlert('formAlert', 'Network error — is the server running?'); btnLoad(btn, false); }
  });
}

/* ── Dashboard ───────────────────────────────────────────── */
function loadDashboard() {
  const u = currentUser; if (!u) return;

  const av = $('avatarInitial');
  if (av) av.textContent = (u.name || u.email || '?')[0].toUpperCase();

  setTxt('userEmail',   u.name || u.email);
  setTxt('userRole',    cap(u.role));
  setTxt('detailEmail', u.email);
  setTxt('detailName',  u.name  || '—');
  setTxt('detailPhone', u.phone || '—');
  setTxt('detailRole',  cap(u.role));

  if (u.role === 'employer') {
    show('freelancerStats', false);
    show('takeTestBtn', false);
    show('postJobBtn');
    show('qaFreelancer', false);
    show('qaEmployer');
    show('tipFreelancer', false);
    show('tipEmployer');
    show('detailVerifyRow', false);
    show('detailScoreRow', false);
  } else {
    setTxt('userScore',  (u.verification_score ?? 0) + '%');
    setTxt('detailScore', (u.verification_score ?? 0) + '%');
    const badge = u.is_verified
      ? '<span class="badge badge-ok">✓ Verified</span>'
      : '<span class="badge badge-no">✗ Not Verified</span>';
    ['verifyBadge', 'detailVerify'].forEach(id => { const el=$(id); if(el) el.innerHTML = badge; });
  }
}

/* ── Home page ───────────────────────────────────────────── */
async function initHomePage() {
  loadHomeStats();
  loadHomeJobs();
}

async function loadHomeStats() {
  try {
    const r    = await fetch(`${API_URL}/stats`);
    const data = await r.json();
    const fmt  = n => n >= 1000 ? (n / 1000).toFixed(1).replace(/\.0$/, '') + 'K+' : n > 0 ? String(n) : '0';
    setTxt('statJobs',        fmt(data.open_jobs   || 0));
    setTxt('statFreelancers', fmt(data.freelancers || 0));
    setTxt('statEmployers',   fmt(data.employers   || 0));
  } catch { /* leave dashes */ }
}

async function loadHomeJobs() {
  const list  = $('homeJobList');
  const empty = $('homeJobEmpty');
  if (!list) return;
  try {
    const r    = await fetch(`${API_URL}/jobs?limit=3&page=1`);
    const data = await r.json();
    const jobs = data.jobs || [];
    list.innerHTML = '';
    if (!jobs.length) { list.style.display = 'none'; if (empty) show('homeJobEmpty'); return; }
    jobs.forEach(job => {
      const card = document.createElement('a');
      card.href  = 'jobs.html';
      card.className = 'home-job-card';
      const budget = job.budget ? `$${Number(job.budget).toLocaleString()}` : 'Budget not listed';
      const tag    = job.category
        ? job.category.charAt(0).toUpperCase() + job.category.slice(1)
        : job.job_type || 'Open';
      card.innerHTML = `
        <div>
          <div class="home-job-title">${esc(job.title)}</div>
          <div class="home-job-meta">
            <span>${esc(job.employer_name || job.employer_email || 'Employer')}</span>
            <span>${budget}</span>
            <span>${job.job_type || 'Remote'}</span>
          </div>
        </div>
        <span class="home-job-tag">${esc(tag)}</span>`;
      list.appendChild(card);
    });
  } catch { list.innerHTML = ''; }
}

/* ── Jobs page ───────────────────────────────────────────── */
function initJobsPage() {
  const toggleBtn = $('postToggle');
  if (currentUser?.role === 'employer' && toggleBtn) {
    show('postToggle');
    toggleBtn.addEventListener('click', togglePostPanel);
    attachPostJobForm();
    attachSkillsInput();
    attachDescCounter();
    attachExpCards();
  }
  loadJobs();
}

function togglePostPanel() {
  const panel = $('postPanel'); if (!panel) return;
  const opening = !panel.classList.contains('show');
  panel.classList.toggle('show', opening);
  const btn = $('postToggle');
  if (btn) btn.textContent = opening ? '✕ Close Form' : '+ Post a Job';
  if (opening) panel.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

function closePostPanel() {
  const panel = $('postPanel'); if (!panel) return;
  panel.classList.remove('show');
  const btn = $('postToggle');
  if (btn) btn.textContent = '+ Post a Job';
}

function attachPostJobForm() {
  const form = $('createJobForm'); if (!form) return;
  form.addEventListener('submit', async e => {
    e.preventDefault();
    if (!validateStep(1)) return;
    const publishBtn = $('publishJobBtn');
    btnLoad(publishBtn, true);

    const experience = document.querySelector('input[name="experience"]:checked')?.value || 'any';
    const budgetRaw  = $('jobBudget')?.value.trim();
    const payload = {
      title:        $('jobTitle')?.value.trim(),
      description:  $('jobDescription')?.value.trim(),
      category:     $('jobCategory')?.value,
      budget:       budgetRaw ? parseFloat(budgetRaw) : null,
      budget_type:  $('jobBudgetType')?.value,
      duration:     $('jobDuration')?.value,
      deadline:     $('jobDeadline')?.value,
      job_type:     $('jobType')?.value,
      location:     $('jobLocation')?.value.trim(),
      skills:       $('jobSkills')?.value,
      experience,
      requirements: $('jobRequirements')?.value.trim(),
    };

    try {
      const r = await fetch(`${API_URL}/jobs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${authToken}` },
        body: JSON.stringify(payload)
      });
      if (r.ok) {
        form.reset();
        const skillsTags = $('skillsTags');
        if (skillsTags) skillsTags.innerHTML = '';
        const hidden = $('jobSkills');
        if (hidden) hidden.value = '';
        window._jobSkills = [];
        show('jobPreview', false);
        const dc = $('descCount'); if (dc) dc.textContent = '0';
        goStep(1);
        closePostPanel();
        loadJobs();
        showSuccessToast('Job published successfully!');
      } else {
        const d = await r.json();
        alert(d.message || 'Failed to post job');
        btnLoad(publishBtn, false);
      }
    } catch { alert('Network error — make sure the server is running.'); btnLoad(publishBtn, false); }
  });
}

function attachSkillsInput() {
  const skillInput = $('jobSkillsInput'); if (!skillInput) return;
  window._jobSkills = [];

  function renderTags() {
    const container = $('skillsTags'); if (!container) return;
    container.innerHTML = window._jobSkills.map((s, i) =>
      `<span class="skill-tag">${esc(s)}<button type="button" onclick="removeSkill(${i})">×</button></span>`
    ).join('');
    const hidden = $('jobSkills');
    if (hidden) hidden.value = window._jobSkills.join(',');
  }

  window.removeSkill = i => { window._jobSkills.splice(i, 1); renderTags(); };

  skillInput.addEventListener('keydown', e => {
    if ((e.key === 'Enter' || e.key === ',') && skillInput.value.trim()) {
      e.preventDefault();
      const val = skillInput.value.replace(/,/g, '').trim();
      if (val && !window._jobSkills.includes(val) && window._jobSkills.length < 10) {
        window._jobSkills.push(val);
        renderTags();
        skillInput.value = '';
      }
    }
  });
}

function attachDescCounter() {
  const desc = $('jobDescription'), counter = $('descCount');
  if (desc && counter) desc.addEventListener('input', () => { counter.textContent = desc.value.length; });
}

function attachExpCards() {
  document.querySelectorAll('.exp-card').forEach(card => {
    card.addEventListener('click', () => {
      document.querySelectorAll('.exp-card').forEach(c => c.classList.remove('on'));
      card.classList.add('on');
    });
  });
}

/* multi-step form */
let currentStep = 1;

function goStep(n) {
  if (n > currentStep && !validateStep(currentStep)) return;
  currentStep = n;
  [1, 2, 3].forEach(i => {
    const s = $(`formStep${i}`);
    if (s) s.style.display = i === n ? '' : 'none';
    const ind = $(`step${i}ind`);
    if (ind) {
      ind.classList.toggle('active', i === n);
      ind.classList.toggle('done', i < n);
    }
  });
}

function validateStep(n) {
  if (n === 1) {
    const title = $('jobTitle')?.value.trim();
    const cat   = $('jobCategory')?.value;
    const desc  = $('jobDescription')?.value.trim();
    if (!title)              { alert('Please enter a job title.'); return false; }
    if (!cat)                { alert('Please select a category.'); return false; }
    if (!desc || desc.length < 30) { alert('Please write a description (at least 30 characters).'); return false; }
  }
  return true;
}

function previewJob() {
  const title  = $('jobTitle')?.value.trim()       || '—';
  const desc   = $('jobDescription')?.value.trim() || '—';
  const cat    = $('jobCategory')?.value || '';
  const budget = $('jobBudget')?.value.trim();
  const btype  = $('jobBudgetType')?.value;
  const dur    = $('jobDuration')?.value;
  const type   = $('jobType')?.value;
  const preview = $('jobPreview'); if (!preview) return;
  $('previewTitle').textContent = title;
  $('previewDesc').textContent  = desc.length > 200 ? desc.slice(0, 200) + '…' : desc;
  const meta = [];
  if (cat)    meta.push(`📂 ${cat}`);
  if (budget) meta.push(`💰 $${budget} (${btype})`);
  if (dur)    meta.push(`⏱ ${dur.replace(/_/g, ' ')}`);
  if (type)   meta.push(`🌍 ${type}`);
  $('previewMeta').textContent = meta.join('  ·  ');
  show('jobPreview');
}

/* ── Jobs list ───────────────────────────────────────────── */
async function loadJobs() {
  const list = $('jobList'); if (!list) return;
  try {
    const r    = await fetch(`${API_URL}/jobs`);
    const data = await r.json();
    const jobs = Array.isArray(data) ? data : (data.jobs ?? []);

    list.innerHTML = '';
    if (!jobs.length) {
      list.innerHTML = `<div class="empty"><div class="empty-ico">💼</div><h3>No jobs yet</h3><p>Be the first to post one!</p></div>`;
      setJobCount(0); return;
    }

    jobs.forEach(job => {
      const card = document.createElement('div');
      card.className = 'job-card';
      const budgetLabel = job.budget ? `💰 $${Number(job.budget).toLocaleString()}` : '';
      const typeIcon    = { remote: '🌍', hybrid: '🏢', onsite: '📍' }[job.job_type] || '';
      const typeLabel   = job.job_type ? `${typeIcon} ${cap(job.job_type)}` : '';
      const catLabel    = job.category ? `📂 ${cap(job.category)}` : '';
      const skillList   = job.skills
        ? job.skills.split(',').filter(Boolean).map(s => `<span class="skill-pill">${esc(s.trim())}</span>`).join('')
        : '';
      card.innerHTML = `
        <div>
          <h3>${esc(job.title)}</h3>
          <p>${esc(job.description)}</p>
          ${skillList ? `<div class="card-skills">${skillList}</div>` : ''}
          <div class="job-meta">
            <span class="job-tag">Open</span>
            ${budgetLabel ? `<span class="job-tag job-tag-budget">${budgetLabel}</span>` : ''}
            ${typeLabel   ? `<span class="job-tag job-tag-type">${typeLabel}</span>`     : ''}
            ${catLabel    ? `<span class="job-tag job-tag-cat">${catLabel}</span>`       : ''}
            <span class="job-date">📅 ${fmtDate(job.created_at)}</span>
          </div>
        </div>
        <div class="job-act">
          ${currentUser?.role === 'freelancer'
            ? `<button class="btn btn-primary btn-sm" onclick="applyJob(${job.id},this)">Apply Now</button>`
            : ''}
        </div>`;
      list.appendChild(card);
    });
    setJobCount(jobs.length);

    const searchInput = $('jobSearch');
    if (searchInput && !searchInput.dataset.wired) {
      searchInput.dataset.wired = 'true';
      searchInput.addEventListener('input', function () {
        const q = this.value.toLowerCase(); let v = 0;
        list.querySelectorAll('.job-card').forEach(c => {
          const show = c.textContent.toLowerCase().includes(q);
          c.style.display = show ? '' : 'none';
          if (show) v++;
        });
        setJobCount(v);
      });
    }
  } catch {
    list.innerHTML = `<div class="empty"><div class="empty-ico">⚠️</div><h3>Could not load jobs</h3><p>Make sure the server is running.</p></div>`;
  }
}

function setJobCount(n) { const el = $('jobCount'); if (el) el.textContent = `${n} job${n !== 1 ? 's' : ''} listed`; }

async function applyJob(id, btn) {
  btnLoad(btn, true);
  try {
    const r = await fetch(`${API_URL}/apply`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${authToken}` },
      body: JSON.stringify({ job_id: id })
    });
    if (r.ok) { btn.textContent = '✓ Applied'; btn.disabled = true; btn.className = 'btn btn-ghost btn-sm'; }
    else { const d = await r.json(); alert(d.message || 'Failed to apply'); btnLoad(btn, false); }
  } catch { alert('Network error'); btnLoad(btn, false); }
}

/* ── Test / My-Jobs page ─────────────────────────────────── */
function setupTestPage() {
  if (!currentUser) {
    show('guestView'); show('employerView', false); show('freelancerView', false); return;
  }

  if (currentUser.role === 'employer') {
    show('employerView'); show('freelancerView', false); show('guestView', false);
    document.title = 'Test — SkillVerify';
    empLoadJobs();
  } else {
    show('freelancerView'); show('employerView', false); show('guestView', false);
    document.title = 'Test — SkillVerify';
    loadMyJobTests();
  }
}

/* ── Freelancer: job-specific tests ─────────────────────── */
async function loadMyJobTests() {
  const list = $('myJobTestList'); if (!list) return;
  list.innerHTML = '<p style="opacity:.5;padding:20px 0 8px">Loading your applications…</p>';
  try {
    const r    = await fetch(`${API_URL}/my-applications`, { headers: { 'Authorization': `Bearer ${authToken}` } });
    const apps = await r.json();
    if (!Array.isArray(apps) || !apps.length) {
      list.innerHTML = `<div class="dcard" style="text-align:center;padding:36px 24px">
        <p style="opacity:.6;margin-bottom:18px">You haven't applied to any jobs yet.</p>
        <a href="jobs.html" class="btn btn-primary">Browse Jobs</a>
      </div>`;
      return;
    }
    list.innerHTML = '';
    apps.forEach(app => {
      const hasTest     = app.question_count > 0;
      const shortlisted = app.status === 'shortlisted';
      const done        = app.job_test_score !== null && app.job_test_score !== undefined;

      let actionHtml;
      if (!hasTest) {
        actionHtml = `<span style="font-size:.82rem;color:var(--t3);white-space:nowrap">No test</span>`;
      } else if (done) {
        actionHtml = `<span style="font-size:.82rem;color:var(--green,#22c55e);white-space:nowrap">Completed</span>`;
      } else if (shortlisted) {
        actionHtml = `<button class="btn btn-primary btn-sm js-take-test">Take Test</button>`;
      } else {
        actionHtml = `<span style="font-size:.82rem;color:var(--t3);white-space:nowrap;text-align:right">Available after<br>shortlisting</span>`;
      }

      const statusColor = shortlisted ? 'var(--green,#22c55e)' : 'var(--blue)';
      const card = document.createElement('div');
      card.className = 'dcard';
      card.style.marginBottom = '14px';
      card.innerHTML = `
        <div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap">
          <div style="flex:1;min-width:0">
            <div style="font-weight:600;margin-bottom:4px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${esc(app.job_title)}</div>
            <div style="font-size:.82rem;color:var(--t3)">
              Status: <span style="color:${statusColor};font-weight:600">${app.status}</span>
              ${done ? ` &nbsp;·&nbsp; Score: <strong style="color:var(--t1)">${app.job_test_score}%</strong>` : ''}
            </div>
          </div>
          ${actionHtml}
        </div>`;

      // Attach listener after DOM insertion to avoid onclick/quote escaping issues
      card.querySelector('.js-take-test')?.addEventListener('click', () => {
        startJobTest(app.job_id, app.job_title);
      });

      list.appendChild(card);
    });
  } catch {
    list.innerHTML = '<p style="color:var(--red)">Failed to load your applications. Try refreshing.</p>';
  }
}

let _activeJobId = null;
async function startJobTest(jobId, title) {
  _activeJobId = jobId;
  show('myJobTestList', false);
  show('jobTestResult', false);
  const form = $('jobQuestionList'); if (!form) return;
  form.innerHTML = '<p style="opacity:.5;padding:16px 0">Loading questions…</p>';
  show('jobTestForm');
  const ttl = $('jobTestTitle');
  if (ttl) ttl.textContent = title;
  try {
    const r         = await fetch(`${API_URL}/job-questions?job_id=${jobId}`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    const questions = await r.json();
    if (!Array.isArray(questions) || !questions.length) {
      form.innerHTML = '<p style="opacity:.6;padding:16px 0">No screening questions set for this job.</p>';
      return;
    }
    renderJobQuestions(questions);
  } catch {
    form.innerHTML = '<p style="color:var(--red)">Error loading questions. Please try again.</p>';
  }
}

function renderJobQuestions(questions) {
  const form = $('jobQuestionList'); if (!form) return;
  form.innerHTML = '';
  questions.forEach((q, i) => {
    const div = document.createElement('div');
    div.className = 'tq';
    div.innerHTML = `
      <div class="tq-num">Question ${i + 1} of ${questions.length}</div>
      <h4>${esc(q.question)}</h4>
      <div class="q-opts">
        ${q.options.map((opt, oi) => `
          <label class="q-opt">
            <input type="radio" name="jq${q.id}" value="${oi}">
            ${esc(opt)}
          </label>`).join('')}
      </div>`;
    form.appendChild(div);
    div.querySelectorAll('.q-opt').forEach(lbl => {
      lbl.addEventListener('click', () => {
        div.querySelectorAll('.q-opt').forEach(l => l.classList.remove('picked'));
        lbl.classList.add('picked');
      });
    });
  });
  const sub = document.createElement('button');
  sub.className = 'btn btn-primary btn-lg'; sub.style.marginTop = '8px';
  sub.textContent = 'Submit Answers →';
  sub.addEventListener('click', () => submitJobTest(questions, sub));
  form.appendChild(sub);
}

async function submitJobTest(questions, btn) {
  const answers = {};
  questions.forEach(q => {
    const sel = document.querySelector(`input[name="jq${q.id}"]:checked`);
    if (sel) answers[q.id] = parseInt(sel.value);
  });
  if (Object.keys(answers).length < questions.length) {
    alert('Please answer all questions before submitting.');
    return;
  }
  btnLoad(btn, true);
  try {
    const r      = await fetch(`${API_URL}/take-job-test?job_id=${_activeJobId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${authToken}` },
      body: JSON.stringify({ answers })
    });
    const result = await r.json();
    show('jobTestForm', false);
    showJobTestResult(result);
  } catch (err) { alert('Error: ' + err.message); btnLoad(btn, false); }
}

function showJobTestResult(result) {
  const el = $('jobTestResult'); if (!el) return;
  show('jobTestResult');
  el.innerHTML = `
    <div class="result-box">
      <div class="score-ring" style="--pct:${result.score}">
        <div class="score-val">${result.score}%</div>
      </div>
      <h3>${result.passed ? '🎉 Great score!' : '📝 Not bad!'}</h3>
      <p>${esc(result.message || '')}</p>
      <p style="opacity:.55;font-size:.85rem">${result.correct} of ${result.total} correct</p>
      <div class="result-btns">
        <button onclick="backToJobList()" class="btn btn-ghost">← My Job Tests</button>
        <a href="dashboard.html" class="btn btn-primary">Dashboard</a>
      </div>
    </div>`;
}

function backToJobList() {
  show('jobTestForm', false);
  show('jobTestResult', false);
  show('myJobTestList');
  _activeJobId = null;
  loadMyJobTests();
}

/* ── Employer: My Jobs ───────────────────────────────────── */
async function empLoadJobs() {
  const list = $('empJobsList'); if (!list) return;
  try {
    const r    = await fetch(`${API_URL}/my-jobs`, { headers: { 'Authorization': `Bearer ${authToken}` } });
    const jobs = await r.json();
    if (!Array.isArray(jobs) || !jobs.length) {
      list.innerHTML = `<div style="padding:20px 0;opacity:.5;text-align:center">
        No jobs posted yet. <a href="jobs.html" style="color:var(--blue)">Post your first job →</a></div>`;
      return;
    }
    list.innerHTML = '';
    jobs.forEach(job => {
      const row = document.createElement('div');
      row.style.cssText = 'display:flex;align-items:center;gap:10px;padding:14px 0;border-bottom:1px solid var(--border);flex-wrap:wrap';
      row.innerHTML = `
        <div style="flex:1;min-width:150px">
          <div style="font-weight:600;font-size:.97rem">${esc(job.title)}</div>
          <div style="font-size:.78rem;opacity:.55;margin-top:2px">
            ${job.application_count} applicant${job.application_count !== 1 ? 's' : ''}
            &nbsp;·&nbsp;
            ${job.is_open
              ? '<span style="color:var(--green)">● Open</span>'
              : '<span style="color:var(--red)">● Closed</span>'}
          </div>
        </div>
        <button class="btn btn-ghost btn-sm" onclick="empViewApplicants(${job.id},\`${esc(job.title)}\`)">
          👥 Applicants${job.application_count ? ` (${job.application_count})` : ''}
        </button>
        <button class="btn btn-ghost btn-sm" onclick="empManageQuestions(${job.id},\`${esc(job.title)}\`)">
          📋 Questions
        </button>`;
      list.appendChild(row);
    });
  } catch {
    list.innerHTML = `<p style="opacity:.5;padding:16px 0">Could not load jobs. Make sure the server is running.</p>`;
  }
}

/* ── Employer: Applicants ────────────────────────────────── */
async function empViewApplicants(jobId, jobTitle) {
  const card   = $('empApplicantsCard');
  const listEl = $('empApplicantsList');
  if (!card || !listEl) return;

  show('empQuestionsCard', false);
  $('empApplicantsTitle').textContent = `Applicants — ${jobTitle}`;
  card.style.display = '';
  listEl.innerHTML = '<p style="opacity:.5;padding:12px 0">Loading…</p>';
  card.scrollIntoView({ behavior: 'smooth', block: 'nearest' });

  try {
    const r    = await fetch(`${API_URL}/job-applications?job_id=${jobId}`, { headers: { 'Authorization': `Bearer ${authToken}` } });
    const apps = await r.json();
    if (!Array.isArray(apps) || !apps.length) {
      listEl.innerHTML = `<p style="opacity:.5;padding:16px 0;text-align:center">No applicants yet for this job.</p>`;
      return;
    }
    listEl.innerHTML = '';

    const hdr = document.createElement('div');
    hdr.style.cssText = 'display:grid;grid-template-columns:1fr auto auto auto;gap:10px;padding:8px 0;border-bottom:2px solid var(--border);font-size:.78rem;font-weight:600;opacity:.5;text-transform:uppercase;letter-spacing:.04em';
    hdr.innerHTML = '<span>Applicant</span><span>Verification</span><span>Test</span><span>Status</span>';
    listEl.appendChild(hdr);

    apps.forEach(app => {
      const row = document.createElement('div');
      row.style.cssText = 'display:grid;grid-template-columns:1fr auto auto auto;gap:10px;align-items:center;padding:12px 0;border-bottom:1px solid var(--border)';
      const verified = app.is_verified
        ? `<span style="font-size:.75rem;padding:3px 9px;border-radius:20px;background:var(--green-dim);color:var(--green)">✓ Verified ${app.verification_score}%</span>`
        : `<span style="font-size:.75rem;padding:3px 9px;border-radius:20px;background:var(--red-dim);color:var(--red)">✗ Not verified</span>`;
      const testScore = app.job_test_score !== null && app.job_test_score !== undefined
        ? `<span style="font-size:.75rem;padding:3px 9px;border-radius:20px;background:var(--surface)">${app.job_test_score}%</span>`
        : `<span style="font-size:.75rem;opacity:.4">—</span>`;
      row.innerHTML = `
        <div>
          <div style="font-weight:600;font-size:.9rem">${esc(app.freelancer_name || app.freelancer_email)}</div>
          <div style="font-size:.78rem;opacity:.55;margin-top:2px">${esc(app.freelancer_email)}</div>
          ${app.cover_note ? `<div style="font-size:.78rem;opacity:.45;margin-top:3px;max-width:300px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">"${esc(app.cover_note)}"</div>` : ''}
          <div style="font-size:.75rem;opacity:.4;margin-top:2px">Applied ${fmtDate(app.created_at)}</div>
        </div>
        <div>${verified}</div>
        <div>${testScore}</div>
        <div>
          <select onchange="empUpdateStatus(${app.id},this)"
            style="padding:5px 8px;border-radius:6px;border:1px solid var(--border);background:var(--bg2);color:var(--t1);font-size:.82rem;cursor:pointer">
            <option value="pending"     ${app.status==='pending'     ? 'selected':''}>Pending</option>
            <option value="shortlisted" ${app.status==='shortlisted' ? 'selected':''}>Shortlisted</option>
            <option value="rejected"    ${app.status==='rejected'    ? 'selected':''}>Rejected</option>
            <option value="hired"       ${app.status==='hired'       ? 'selected':''}>Hired ✓</option>
          </select>
        </div>`;
      listEl.appendChild(row);
    });
  } catch {
    listEl.innerHTML = `<p style="opacity:.5;padding:12px 0">Could not load applicants.</p>`;
  }
}

function empCloseApplicants() { show('empApplicantsCard', false); }

async function empUpdateStatus(appId, sel) {
  const status = sel.value;
  try {
    const r = await fetch(`${API_URL}/application-status?id=${appId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${authToken}` },
      body: JSON.stringify({ status })
    });
    if (r.ok) showSuccessToast(`Status updated to "${cap(status)}" ✓`);
    else { const d = await r.json(); alert(d.message || 'Update failed'); }
  } catch { alert('Network error'); }
}

/* ── Employer: Screening questions ───────────────────────── */
let _empQJobId = null;
let _empQs     = [];

async function empManageQuestions(jobId, jobTitle) {
  _empQJobId = jobId;
  show('empApplicantsCard', false);
  const card = $('empQuestionsCard'); if (!card) return;
  card.style.display = '';
  $('empQuestionsTitle').textContent = `Screening Questions — ${jobTitle}`;
  card.scrollIntoView({ behavior: 'smooth', block: 'nearest' });

  // Use the employer endpoint so correct_idx is returned from the database
  try {
    const r    = await fetch(`${API_URL}/job-questions-employer?job_id=${jobId}`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    const data = await r.json();
    _empQs = Array.isArray(data) && data.length
      ? data.map(q => ({ question: q.question, options: q.options || ['', ''], correct_idx: q.correct_idx != null ? q.correct_idx : 0 }))
      : [{ question: '', options: ['', ''], correct_idx: 0 }];
  } catch { _empQs = [{ question: '', options: ['', ''], correct_idx: 0 }]; }

  empRenderQuestions();
  empLoadApplicantScores(jobId);
}

function empCloseQuestions() {
  show('empQuestionsCard', false);
  _empQJobId = null;
  _empQs = [];
  const sc = $('empApplicantScores');
  if (sc) sc.innerHTML = '<p style="opacity:.45;font-size:.85rem">Select a job to view results.</p>';
}

async function empLoadApplicantScores(jobId) {
  const wrap = $('empApplicantScores');
  if (!wrap) return;
  wrap.innerHTML = '<p style="opacity:.5;font-size:.85rem">Loading applicant scores…</p>';
  try {
    const r    = await fetch(`${API_URL}/job-applications?job_id=${jobId}`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    const apps = await r.json();
    if (!Array.isArray(apps) || !apps.length) {
      wrap.innerHTML = '<p style="opacity:.45;font-size:.85rem">No applicants yet for this job.</p>';
      return;
    }
    const scored  = apps.filter(a => a.job_test_score !== null && a.job_test_score !== undefined)
                        .sort((a, b) => b.job_test_score - a.job_test_score);
    const pending = apps.filter(a => a.job_test_score === null || a.job_test_score === undefined);
    let html = '';
    if (scored.length) {
      html += `<div style="font-size:.75rem;font-weight:600;opacity:.45;text-transform:uppercase;letter-spacing:.04em;margin-bottom:10px">${scored.length} of ${apps.length} applicant${apps.length !== 1 ? 's' : ''} completed the test</div>`;
      scored.forEach((app, i) => {
        const score  = app.job_test_score;
        const passed = score >= 60;
        const color  = passed ? 'var(--green)' : 'var(--red)';
        const bg     = passed ? 'var(--green-dim)' : 'var(--red-dim)';
        html += `
          <div style="display:flex;align-items:center;gap:10px;padding:9px 0;border-bottom:1px solid var(--border)">
            <span style="font-size:.75rem;font-weight:700;opacity:.3;min-width:20px">#${i + 1}</span>
            <div style="flex:1;min-width:0">
              <div style="font-weight:600;font-size:.88rem;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${esc(app.freelancer_name || app.freelancer_email)}</div>
              <div style="font-size:.75rem;opacity:.5">${esc(app.freelancer_email)}</div>
            </div>
            <span style="font-size:.82rem;padding:3px 10px;border-radius:20px;background:${bg};color:${color};font-weight:600;white-space:nowrap">${score}%</span>
            <span style="font-size:.75rem;opacity:.5;white-space:nowrap">${cap(app.status)}</span>
          </div>`;
      });
    }
    if (pending.length) {
      html += `<p style="font-size:.8rem;opacity:.4;margin-top:${scored.length ? '10px' : '0'}">${pending.length} applicant${pending.length !== 1 ? 's' : ''} have not taken the test yet.</p>`;
    }
    if (!scored.length && !pending.length) {
      html = '<p style="opacity:.45;font-size:.85rem">No applicants yet.</p>';
    }
    wrap.innerHTML = html;
  } catch {
    wrap.innerHTML = '<p style="opacity:.45;font-size:.85rem">Could not load applicant scores.</p>';
  }
}

function empRenderQuestions() {
  const wrap = $('empQuestionsList'); if (!wrap) return;
  wrap.innerHTML = '';

  _empQs.forEach((q, qi) => {
    const block = document.createElement('div');
    block.style.cssText = 'background:var(--bg2);border:1px solid var(--border);border-radius:12px;padding:16px;display:flex;flex-direction:column;gap:10px';

    const qHdr = document.createElement('div');
    qHdr.style.cssText = 'display:flex;align-items:center;gap:8px';
    qHdr.innerHTML = `
      <span style="font-size:.78rem;font-weight:700;opacity:.4;min-width:24px">Q${qi + 1}</span>
      <input type="text" value="${esc(q.question)}" placeholder="Type your question here…"
        oninput="_empQs[${qi}].question=this.value"
        style="flex:1;padding:8px 12px;border-radius:8px;border:1px solid var(--border);background:var(--bg);color:var(--t1);font-size:.9rem">
      ${_empQs.length > 1
        ? `<button onclick="empRemoveQ(${qi})" style="background:none;border:none;cursor:pointer;color:var(--red);font-size:1.1rem;padding:0 4px" title="Remove question">✕</button>`
        : ''}`;
    block.appendChild(qHdr);

    const hint = document.createElement('p');
    hint.style.cssText = 'font-size:.75rem;opacity:.45;margin:0 0 2px 32px';
    hint.textContent = 'Fill in each option, then select (●) the correct answer.';
    block.appendChild(hint);

    const optsWrap = document.createElement('div');
    optsWrap.style.cssText = 'display:flex;flex-direction:column;gap:7px;padding-left:32px';
    q.options.forEach((opt, oi) => {
      const optRow = document.createElement('div');
      optRow.style.cssText = 'display:flex;align-items:center;gap:8px';
      optRow.innerHTML = `
        <input type="radio" name="correct_${qi}" value="${oi}" ${q.correct_idx === oi ? 'checked' : ''}
          onchange="_empQs[${qi}].correct_idx=${oi}"
          style="width:16px;height:16px;cursor:pointer;accent-color:var(--green)">
        <input type="text" value="${esc(opt)}" placeholder="Option ${oi + 1}"
          oninput="_empQs[${qi}].options[${oi}]=this.value"
          style="flex:1;padding:6px 10px;border-radius:7px;border:1px solid var(--border);background:var(--bg);color:var(--t1);font-size:.86rem">
        ${q.options.length > 2
          ? `<button onclick="empRemoveOpt(${qi},${oi})" style="background:none;border:none;cursor:pointer;color:var(--red);font-size:.95rem;padding:0 2px" title="Remove option">✕</button>`
          : ''}`;
      optsWrap.appendChild(optRow);
    });
    block.appendChild(optsWrap);

    if (q.options.length < 6) {
      const addOpt = document.createElement('button');
      addOpt.className = 'btn btn-ghost btn-sm';
      addOpt.style.cssText = 'align-self:flex-start;margin-left:32px;font-size:.8rem';
      addOpt.textContent = '+ Add option';
      addOpt.onclick = () => empAddOpt(qi);
      block.appendChild(addOpt);
    }
    wrap.appendChild(block);
  });

  const addBtn = $('empAddQBtn');
  if (addBtn) addBtn.style.display = _empQs.length >= 10 ? 'none' : '';
}

function empAddQuestion() {
  if (_empQs.length >= 10) return;
  _empQs.push({ question: '', options: ['', ''], correct_idx: 0 });
  empRenderQuestions();
  $('empQuestionsList')?.lastElementChild?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

function empRemoveQ(qi) {
  _empQs.splice(qi, 1);
  if (!_empQs.length) _empQs = [{ question: '', options: ['', ''], correct_idx: 0 }];
  empRenderQuestions();
}

function empAddOpt(qi) {
  if (_empQs[qi].options.length >= 6) return;
  _empQs[qi].options.push('');
  empRenderQuestions();
}

function empRemoveOpt(qi, oi) {
  if (_empQs[qi].options.length <= 2) return;
  _empQs[qi].options.splice(oi, 1);
  if (_empQs[qi].correct_idx >= _empQs[qi].options.length) _empQs[qi].correct_idx = 0;
  empRenderQuestions();
}

async function empSaveQuestions() {
  if (!_empQJobId) return;
  for (const q of _empQs) {
    if (!q.question.trim()) { alert('Please fill in all question texts before saving.'); return; }
    for (const o of q.options) { if (!o.trim()) { alert('Please fill in all answer options before saving.'); return; } }
  }
  try {
    const r = await fetch(`${API_URL}/job-questions?job_id=${_empQJobId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${authToken}` },
      body: JSON.stringify({ questions: _empQs })
    });
    if (r.ok) {
      const msg = $('empQSavedMsg');
      if (msg) { msg.style.display = 'inline'; setTimeout(() => msg.style.display = 'none', 2500); }
      showSuccessToast('Screening questions saved!');
    } else {
      const d = await r.json(); alert(d.message || 'Failed to save questions');
    }
  } catch { alert('Network error — make sure the server is running.'); }
}

/* ═══════════════════════════════════════════════════════════
   ADMIN PAGE
════════════════════════════════════════════════════════════ */
let _adminAllUsers = [];
let _adminFilterRole = 'all';

function initAdminPage() {
  if (!currentUser || currentUser.role !== 'admin') {
    show('adminView', false);
    show('adminDenied');
    return;
  }
  show('adminDenied', false);
  show('adminView');
  adminLoadStats();
  adminLoadUsers();
  adminLoadJobs();
}

async function adminLoadStats() {
  try {
    const r = await fetch(`${API_URL}/admin/stats`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    const d = await r.json();
    setTxt('aStat0', d.total_users  ?? '—');
    setTxt('aStat1', d.freelancers  ?? '—');
    setTxt('aStat2', d.employers    ?? '—');
    setTxt('aStat3', d.verified     ?? '—');
    setTxt('aStat4', d.open_jobs    ?? '—');
    setTxt('aStat5', d.applications ?? '—');
  } catch { /* leave dashes */ }
}

async function adminLoadUsers() {
  const wrap = $('adminUsersList'); if (!wrap) return;
  wrap.innerHTML = '<p style="opacity:.5;padding:12px 0">Loading…</p>';
  try {
    const r = await fetch(`${API_URL}/admin/users`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    _adminAllUsers = await r.json();
    if (!Array.isArray(_adminAllUsers)) { _adminAllUsers = []; }
    adminRenderUsers();
  } catch {
    wrap.innerHTML = '<p style="opacity:.5;padding:12px 0">Could not load users.</p>';
  }
}

function adminFilterUsers(role, btn) {
  _adminFilterRole = role;
  document.querySelectorAll('.admin-filter-btn').forEach(b => b.classList.remove('active'));
  if (btn) btn.classList.add('active');
  const search = $('adminUserSearch')?.value || '';
  adminRenderUsers(search);
}

function adminSearchUsers(q) {
  adminRenderUsers(q);
}

function adminRenderUsers(search = '') {
  const wrap = $('adminUsersList'); if (!wrap) return;
  const q = search.toLowerCase();
  const list = _adminAllUsers.filter(u => {
    if (_adminFilterRole !== 'all' && u.role !== _adminFilterRole) return false;
    if (q && !u.name.toLowerCase().includes(q) && !u.email.toLowerCase().includes(q)) return false;
    return true;
  });

  if (!list.length) {
    wrap.innerHTML = '<p style="opacity:.45;padding:16px 0;text-align:center">No users match the filter.</p>';
    return;
  }

  const hdr = document.createElement('div');
  hdr.className = 'admin-user-row';
  hdr.style.cssText = 'font-size:.72rem;font-weight:700;opacity:.4;text-transform:uppercase;letter-spacing:.05em;padding-bottom:6px;border-bottom:2px solid var(--border)';
  hdr.innerHTML = '<span></span><span>User</span><span class="admin-role-col">Role</span><span class="admin-score">Score</span><span>Action</span>';
  wrap.innerHTML = '';
  wrap.appendChild(hdr);

  list.forEach(u => {
    const row = document.createElement('div');
    row.className = 'admin-user-row';

    const initial = (u.name || u.email || '?')[0].toUpperCase();
    const roleBadge = u.role === 'freelancer'
      ? `<span style="padding:2px 8px;border-radius:20px;background:var(--blue-dim);color:var(--blue);font-size:.72rem;font-weight:700">Freelancer</span>`
      : u.role === 'employer'
      ? `<span style="padding:2px 8px;border-radius:20px;background:rgba(168,85,247,.12);color:var(--indigo);font-size:.72rem;font-weight:700">Employer</span>`
      : `<span style="padding:2px 8px;border-radius:20px;background:var(--surface2);color:var(--t2);font-size:.72rem;font-weight:700">Admin</span>`;

    const verifiedBadge = u.is_verified
      ? `<span style="font-size:.72rem;padding:2px 8px;border-radius:20px;background:var(--green-dim);color:var(--green);font-weight:700">✓ ${u.verification_score}%</span>`
      : `<span style="font-size:.72rem;opacity:.4">—</span>`;

    let actionHtml = '<span style="font-size:.75rem;opacity:.3">—</span>';
    if (u.role === 'freelancer') {
      if (u.is_verified) {
        actionHtml = `<button class="btn btn-sm" style="background:var(--red-dim);color:var(--red);border:1px solid rgba(248,113,113,.2)"
          onclick="adminSetVerified(${u.id}, false, this)">Unverify</button>`;
      } else {
        actionHtml = `<button class="btn btn-primary btn-sm"
          onclick="adminSetVerified(${u.id}, true, this)">Verify</button>`;
      }
    }

    row.innerHTML = `
      <div class="admin-avatar">${esc(initial)}</div>
      <div style="min-width:0">
        <div style="font-weight:600;font-size:.88rem;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${esc(u.name || '—')}</div>
        <div style="font-size:.76rem;color:var(--t3);white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${esc(u.email)}</div>
        ${u.phone ? `<div style="font-size:.73rem;color:var(--t3)">${esc(u.phone)}</div>` : ''}
      </div>
      <div class="admin-role-col">${roleBadge}</div>
      <div class="admin-score">${verifiedBadge}</div>
      <div>${actionHtml}</div>`;
    wrap.appendChild(row);
  });
}

async function adminSetVerified(userId, verified, btn) {
  btn.disabled = true;
  btn.textContent = '…';
  try {
    const r = await fetch(`${API_URL}/admin/verify-user?id=${userId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${authToken}` },
      body: JSON.stringify({ verified })
    });
    const d = await r.json();
    if (r.ok) {
      showSuccessToast(d.message || 'Updated');
      // refresh list and stats
      await adminLoadUsers();
      adminLoadStats();
    } else {
      alert(d.message || 'Update failed');
      btn.disabled = false;
      btn.textContent = verified ? 'Verify' : 'Unverify';
    }
  } catch {
    alert('Network error');
    btn.disabled = false;
    btn.textContent = verified ? 'Verify' : 'Unverify';
  }
}

async function adminLoadJobs() {
  const wrap = $('adminJobsList'); if (!wrap) return;
  wrap.innerHTML = '<p style="opacity:.5;padding:12px 0">Loading…</p>';
  try {
    const r    = await fetch(`${API_URL}/admin/jobs`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    const jobs = await r.json();
    if (!Array.isArray(jobs) || !jobs.length) {
      wrap.innerHTML = '<p style="opacity:.45;padding:16px 0;text-align:center">No jobs yet.</p>';
      setTxt('adminJobCount', '0 jobs');
      return;
    }
    setTxt('adminJobCount', `${jobs.length} job${jobs.length !== 1 ? 's' : ''}`);

    const hdr = document.createElement('div');
    hdr.className = 'admin-job-row';
    hdr.style.cssText = 'font-size:.72rem;font-weight:700;opacity:.4;text-transform:uppercase;letter-spacing:.05em;padding-bottom:6px;border-bottom:2px solid var(--border)';
    hdr.innerHTML = '<span>Job</span><span class="admin-emp">Employer</span><span class="admin-app-count">Apps</span><span>Status</span>';
    wrap.innerHTML = '';
    wrap.appendChild(hdr);

    jobs.forEach(job => {
      const row = document.createElement('div');
      row.className = 'admin-job-row';
      const statusBadge = job.is_open
        ? `<span style="padding:2px 9px;border-radius:20px;background:var(--green-dim);color:var(--green);font-size:.72rem;font-weight:700">Open</span>`
        : `<span style="padding:2px 9px;border-radius:20px;background:var(--red-dim);color:var(--red);font-size:.72rem;font-weight:700">Closed</span>`;
      row.innerHTML = `
        <div style="min-width:0">
          <div style="font-weight:600;font-size:.88rem;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${esc(job.title)}</div>
          <div style="font-size:.74rem;color:var(--t3)">${fmtDate(job.created_at)}</div>
        </div>
        <div class="admin-emp" style="min-width:0">
          <div style="font-size:.84rem;font-weight:500;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${esc(job.employer_name || job.employer_email)}</div>
          <div style="font-size:.73rem;color:var(--t3)">${esc(job.employer_email)}</div>
        </div>
        <div class="admin-app-count" style="font-size:.84rem;font-weight:700;text-align:center">${job.application_count}</div>
        <div>${statusBadge}</div>`;
      wrap.appendChild(row);
    });
  } catch {
    wrap.innerHTML = '<p style="opacity:.45;padding:12px 0">Could not load jobs.</p>';
  }
}
