const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';

async function hydrationTest() {
  console.log('🔍 Svelte Hydration Test\n');
  
  const browser = await chromium.launch({ headless: false, slowMo: 300 });
  const context = await browser.newContext({ viewport: { width: 1200, height: 800 } });
  const page = await context.newPage();

  // Capture console
  page.on('console', msg => {
    const text = msg.text();
    if (text.includes('error') || text.includes('Error') || text.includes('hydrat')) {
      console.log(`[Console]: ${text.substring(0, 300)}`);
    }
  });

  page.on('pageerror', err => {
    console.log(`[Page Error]: ${err.message}`);
  });

  try {
    console.log('1. Loading page...');
    await page.goto(BASE_URL, { waitUntil: 'load' });
    await page.waitForTimeout(3000);
    
    // Check if Svelte is loaded
    console.log('\n2. Checking for Svelte app...');
    const svelteCheck = await page.evaluate(() => {
      // Check for Svelte internal markers
      const hasSvelteData = document.body.innerHTML.includes('svelte');
      const hasHydrationMarker = document.querySelector('[data-svelte-h]') !== null;
      
      // Check if any element has Svelte event handlers
      const allElements = document.querySelectorAll('*');
      let hasEventListeners = false;
      
      // We can't easily check for event listeners, but we can check if the app is interactive
      return {
        hasSvelteData,
        hasHydrationMarker,
        bodyLength: document.body.innerHTML.length,
        scriptsLoaded: document.scripts.length
      };
    });
    console.log(`   Svelte check: ${JSON.stringify(svelteCheck)}`);

    // Test other interactive elements
    console.log('\n3. Testing password visibility toggle...');
    
    // First fill in username field to see if inputs work
    const usernameInput = page.locator('input[autocomplete="username"]');
    await usernameInput.fill('testuser');
    const usernameValue = await usernameInput.inputValue();
    console.log(`   Username input works: ${usernameValue === 'testuser'}`);
    
    // Try clicking the eye icon to toggle password visibility
    const passwordInput = page.locator('input[type="password"], input[type="text"]').nth(1);
    const initialType = await passwordInput.getAttribute('type');
    console.log(`   Password initial type: ${initialType}`);
    
    // Find and click the eye button
    const eyeBtn = page.locator('button:has(svg)').filter({ has: page.locator('svg[class*="Eye"]') }).first();
    const eyeBtnExists = await eyeBtn.count() > 0;
    console.log(`   Eye button exists: ${eyeBtnExists}`);
    
    if (eyeBtnExists) {
      await eyeBtn.click();
      await page.waitForTimeout(500);
      const newType = await passwordInput.getAttribute('type');
      console.log(`   Password type after click: ${newType}`);
      console.log(`   Password toggle works: ${initialType !== newType}`);
    }

    // Test language toggle
    console.log('\n4. Testing language toggle...');
    const langBtn = page.locator('button:has-text("EN"), button:has-text("ES")');
    const beforeLang = await langBtn.textContent();
    console.log(`   Current language: ${beforeLang}`);
    
    await langBtn.click();
    await page.waitForTimeout(500);
    
    const afterLang = await langBtn.textContent();
    console.log(`   After click: ${afterLang}`);
    console.log(`   Language toggle works: ${beforeLang !== afterLang}`);

    // Test form submission
    console.log('\n5. Testing form submit...');
    await page.fill('input[autocomplete="username"]', 'admin');
    await page.fill('input[type="password"]', 'wrongpassword');
    
    // Click sign in
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    
    // Check for error message
    const errorVisible = await page.locator('.text-stopped, [class*="error"]').isVisible().catch(() => false);
    console.log(`   Error message appears: ${errorVisible}`);

    // Now test forgot password again
    console.log('\n6. Testing forgot password state change...');
    
    // Re-navigate to clear any state
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    const forgotBtn = page.locator('button').filter({ hasText: 'Forgot password?' });
    const forgotExists = await forgotBtn.count() > 0;
    console.log(`   Forgot button found: ${forgotExists}`);
    
    if (forgotExists) {
      // Get button info
      const btnInfo = await forgotBtn.evaluate((btn) => {
        return {
          onclick: btn.onclick,
          hasClickHandler: typeof btn.onclick === 'function',
          listeners: Object.keys(btn).filter(k => k.startsWith('__'))
        };
      });
      console.log(`   Button info: ${JSON.stringify(btnInfo)}`);
      
      await forgotBtn.click();
      await page.waitForTimeout(2000);
      
      await page.screenshot({ path: 'test-screenshots/hydration-after-forgot.png' });
      
      // Check view state
      const hasBackBtn = await page.locator('text=Back to login').isVisible().catch(() => false);
      const hasSendCode = await page.locator('button').filter({ hasText: /Send|Enviar/ }).count() > 0;
      console.log(`   Back button visible: ${hasBackBtn}`);
      console.log(`   Send button visible: ${hasSendCode}`);
    }

  } catch (error) {
    console.error('\n💥 ERROR:', error.message);
  } finally {
    console.log('\n⏳ Browser open for 15s...');
    await page.waitForTimeout(15000);
    await browser.close();
  }
}

hydrationTest().catch(console.error);
