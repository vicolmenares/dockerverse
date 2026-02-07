const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';

async function debugTest() {
  console.log('🔍 Starting Debug Test...\n');
  
  const browser = await chromium.launch({ 
    headless: false,
    slowMo: 1000
  });
  
  const page = await browser.newPage();
  
  // Log all console messages
  page.on('console', msg => console.log(`[Console ${msg.type()}]: ${msg.text()}`));
  page.on('request', req => console.log(`[Request]: ${req.method()} ${req.url()}`));
  page.on('response', res => console.log(`[Response]: ${res.status()} ${res.url()}`));

  try {
    console.log('\n📍 Step 1: Navigate to app');
    await page.goto(BASE_URL, { waitUntil: 'networkidle' });
    await page.screenshot({ path: 'test-screenshots/debug-01-initial.png', fullPage: true });
    
    console.log('\n📍 Step 2: Check what we see');
    const pageContent = await page.content();
    console.log('Page title:', await page.title());
    
    // List all visible buttons
    const buttons = await page.locator('button').all();
    console.log(`Found ${buttons.length} buttons`);
    for (let i = 0; i < Math.min(buttons.length, 5); i++) {
      const text = await buttons[i].textContent();
      console.log(`  Button ${i}: "${text.trim()}"`);
    }
    
    // List all inputs
    const inputs = await page.locator('input').all();
    console.log(`Found ${inputs.length} inputs`);
    for (let i = 0; i < inputs.length; i++) {
      const type = await inputs[i].getAttribute('type');
      const name = await inputs[i].getAttribute('name');
      const placeholder = await inputs[i].getAttribute('placeholder');
      console.log(`  Input ${i}: type=${type}, name=${name}, placeholder=${placeholder}`);
    }
    
    console.log('\n📍 Step 3: Attempt login');
    // Find username input more specifically
    const usernameInput = page.locator('input').first();
    await usernameInput.fill('admin');
    
    const passwordInput = page.locator('input[type="password"]');
    await passwordInput.fill('admin123');
    
    await page.screenshot({ path: 'test-screenshots/debug-02-filled.png', fullPage: true });
    
    // Find and click submit
    const submitBtn = page.locator('button[type="submit"]');
    console.log('Submit button visible:', await submitBtn.isVisible());
    await submitBtn.click();
    
    // Wait for navigation/response
    await page.waitForTimeout(3000);
    await page.screenshot({ path: 'test-screenshots/debug-03-after-login.png', fullPage: true });
    
    console.log('\n📍 Step 4: Check current state after login');
    console.log('Current URL:', page.url());
    
    // Check localStorage
    const localStorage = await page.evaluate(() => {
      const items = {};
      for (let i = 0; i < window.localStorage.length; i++) {
        const key = window.localStorage.key(i);
        items[key] = window.localStorage.getItem(key).substring(0, 50) + '...';
      }
      return items;
    });
    console.log('LocalStorage:', JSON.stringify(localStorage, null, 2));
    
    // List visible text that indicates we're logged in
    const h1s = await page.locator('h1, h2, h3').allTextContents();
    console.log('Headers found:', h1s.join(', '));
    
    console.log('\n📍 Step 5: Try to find header elements');
    const header = page.locator('header, nav, [class*="header"]').first();
    const headerHTML = await header.innerHTML().catch(() => 'Not found');
    console.log('Header content preview:', headerHTML.substring(0, 300) + '...');
    
    // Check for user menu / dropdown
    const relativeButtons = await page.locator('.relative button, button.relative').all();
    console.log(`Found ${relativeButtons.length} relative buttons (potential dropdown triggers)`);
    
    console.log('\n📍 Step 6: Page refresh test');
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    await page.screenshot({ path: 'test-screenshots/debug-04-after-refresh.png', fullPage: true });
    
    // Check if we're still logged in
    const loginFormVisible = await page.locator('input[type="password"]').isVisible();
    console.log('Login form visible after refresh:', loginFormVisible);
    
    if (!loginFormVisible) {
      console.log('✅ SESSION PERSISTED!');
    } else {
      console.log('❌ SESSION LOST - back to login');
    }
    
  } catch (error) {
    console.log('\n❌ Error:', error.message);
    await page.screenshot({ path: 'test-screenshots/debug-error.png', fullPage: true });
  }

  console.log('\n🏁 Debug test complete. Check screenshots in test-screenshots/');
  
  // Keep browser open for manual inspection
  console.log('Browser will stay open for 30 seconds for manual inspection...');
  await page.waitForTimeout(30000);
  
  await browser.close();
}

debugTest().catch(console.error);
