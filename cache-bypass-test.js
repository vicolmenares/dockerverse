const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';

async function cacheBypassTest() {
  console.log('🔍 Cache Bypass Forgot Password Test\n');
  
  const browser = await chromium.launch({ headless: false, slowMo: 400 });
  // Create context with no cache
  const context = await browser.newContext({
    viewport: { width: 1200, height: 800 },
    bypassCSP: true,
    ignoreHTTPSErrors: true,
  });
  
  const page = await context.newPage();

  // Clear all caches before starting
  await context.clearCookies();
  
  // Capture console
  page.on('console', msg => {
    console.log(`[${msg.type()}]: ${msg.text()}`);
  });

  try {
    // Hard refresh with cache bypass
    console.log('1. Hard refresh with no cache...');
    await page.goto(BASE_URL + '?nocache=' + Date.now(), { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    // Check the source of the JS to verify it's the new version
    console.log('\n2. Checking if forgotView is in the page bundle...');
    const hasForgotView = await page.evaluate(() => {
      // Check if there's any evidence of forgot password in the app
      const scripts = Array.from(document.scripts);
      const scriptContents = scripts.map(s => s.src).filter(Boolean);
      return {
        scriptUrls: scriptContents,
        pageSource: document.body.innerHTML.includes('Forgot') || document.body.innerHTML.includes('Olvidaste')
      };
    });
    console.log(`   Scripts: ${JSON.stringify(hasForgotView.scriptUrls.slice(0, 3))}`);
    console.log(`   Has Forgot text: ${hasForgotView.pageSource}`);

    // Find the forgot link
    console.log('\n3. Finding "Forgot password?" button...');
    const forgotBtn = page.locator('button').filter({ hasText: /Forgot|Olvidaste/ }).first();
    const exists = await forgotBtn.count() > 0;
    console.log(`   Found: ${exists}`);
    
    if (exists) {
      // Get button HTML
      const btnHtml = await forgotBtn.evaluate(el => el.outerHTML);
      console.log(`   Button HTML: ${btnHtml.substring(0, 200)}`);
      
      // Check what happens when we click
      console.log('\n4. Attaching mutation observer...');
      await page.evaluate(() => {
        window.__mutations = [];
        const observer = new MutationObserver((mutations) => {
          mutations.forEach((m) => {
            if (m.type === 'childList' && m.addedNodes.length > 0) {
              window.__mutations.push('Added: ' + m.target.className);
            }
            if (m.type === 'childList' && m.removedNodes.length > 0) {
              window.__mutations.push('Removed: ' + m.target.className);
            }
          });
        });
        observer.observe(document.body, { childList: true, subtree: true });
      });
      
      console.log('\n5. Clicking button...');
      await forgotBtn.click();
      await page.waitForTimeout(3000);
      
      // Check mutations
      const mutations = await page.evaluate(() => window.__mutations);
      console.log(`   Mutations detected: ${mutations.length}`);
      if (mutations.length > 0) {
        console.log(`   First few: ${JSON.stringify(mutations.slice(0, 5))}`);
      }
      
      // Check visible buttons
      const buttons = await page.evaluate(() => {
        return [...document.querySelectorAll('button')].map(b => b.textContent.trim()).filter(t => t.length > 0);
      });
      console.log(`\n6. Visible buttons: ${JSON.stringify(buttons)}`);
      
      // Check if "Back to login" or "Send Code" exists
      const backBtn = await page.locator('button').filter({ hasText: /Back|Volver/ }).count();
      const sendBtn = await page.locator('button').filter({ hasText: /Send|Enviar/ }).count();
      console.log(`   Back buttons: ${backBtn}, Send buttons: ${sendBtn}`);
      
      await page.screenshot({ path: 'test-screenshots/cache-bypass-result.png' });
      
      // Check the current view by looking at the DOM
      console.log('\n7. Checking view state...');
      const viewState = await page.evaluate(() => {
        // Look for unique elements in each view
        const hasLoginForm = !!document.querySelector('form button[type="submit"]');
        const hasBackBtn = document.body.textContent.includes('Back to login') || document.body.textContent.includes('Volver');
        const hasSendCode = document.body.textContent.includes('Send Code') || document.body.textContent.includes('Enviar Código');
        const hasCodeInput = !!document.querySelector('input[maxlength="6"]');
        
        return { hasLoginForm, hasBackBtn, hasSendCode, hasCodeInput };
      });
      console.log(`   View state: ${JSON.stringify(viewState)}`);
    }

  } catch (error) {
    console.error('\n💥 ERROR:', error.message);
    await page.screenshot({ path: 'test-screenshots/cache-bypass-error.png' });
  } finally {
    console.log('\n⏳ Browser open for 15s...');
    await page.waitForTimeout(15000);
    await browser.close();
  }
}

cacheBypassTest().catch(console.error);
