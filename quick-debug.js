const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';

async function debugTest() {
  console.log('🔍 Debug Forgot Password Test\n');
  
  const browser = await chromium.launch({ headless: false, slowMo: 300 });
  const context = await browser.newContext({ viewport: { width: 1200, height: 800 } });
  const page = await context.newPage();

  // Capture ALL console messages
  page.on('console', msg => {
    console.log(`[Console ${msg.type()}]: ${msg.text()}`);
  });

  try {
    console.log('1. Loading page...');
    await page.goto(BASE_URL + '?t=' + Date.now(), { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);
    
    console.log('\n2. Clicking Forgot Password...');
    const forgotBtn = page.locator('button').filter({ hasText: /Forgot|Olvidaste/ }).first();
    await forgotBtn.click();
    await page.waitForTimeout(2000);
    
    console.log('\n3. Checking result...');
    const buttons = await page.evaluate(() => {
      return [...document.querySelectorAll('button')].map(b => b.textContent.trim()).filter(t => t.length > 0 && t.length < 30);
    });
    console.log(`   Visible buttons: ${JSON.stringify(buttons)}`);
    
    await page.screenshot({ path: 'test-screenshots/debug-final.png' });

  } catch (error) {
    console.error('\n💥 ERROR:', error.message);
  } finally {
    console.log('\n⏳ Browser open for 10s...');
    await page.waitForTimeout(10000);
    await browser.close();
  }
}

debugTest().catch(console.error);
