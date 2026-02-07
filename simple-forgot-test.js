const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';

async function simpleForgotTest() {
  console.log('🔍 Simple Forgot Password Test\n');
  
  const browser = await chromium.launch({ headless: false, slowMo: 500 });
  const page = await browser.newPage({ viewport: { width: 1200, height: 800 } });

  // Capture ALL console messages
  page.on('console', msg => {
    console.log(`[Console ${msg.type()}]: ${msg.text()}`);
  });

  page.on('pageerror', err => {
    console.log(`[Page Error]: ${err.message}`);
  });

  try {
    console.log('1. Go to login page...');
    await page.goto(BASE_URL, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    // Check current page state
    console.log('\n2. Current page HTML check:');
    const loginFormVisible = await page.locator('form').isVisible();
    console.log(`   Form visible: ${loginFormVisible}`);
    
    // Take screenshot before
    await page.screenshot({ path: 'test-screenshots/simple-01-before-click.png' });
    
    // Find and click forgot password using evaluate
    console.log('\n3. Clicking Forgot Password link via JavaScript...');
    const html = await page.evaluate(() => {
      // Find all buttons with "Forgot" text
      const buttons = [...document.querySelectorAll('button')];
      const forgotBtn = buttons.find(b => b.textContent.includes('Forgot') || b.textContent.includes('Olvidaste'));
      if (forgotBtn) {
        console.log('Found forgot button:', forgotBtn.textContent);
        forgotBtn.click();
        return 'clicked';
      }
      return 'not found';
    });
    console.log(`   Result: ${html}`);
    
    await page.waitForTimeout(2000);
    await page.screenshot({ path: 'test-screenshots/simple-02-after-click.png' });
    
    // Check what's visible now
    console.log('\n4. After click - checking visible elements:');
    const allButtonTexts = await page.evaluate(() => {
      return [...document.querySelectorAll('button')].map(b => b.textContent.trim());
    });
    console.log(`   Buttons: ${JSON.stringify(allButtonTexts)}`);
    
    // Check if we have the "Back to login" element
    const backToLoginVisible = await page.locator('button:has-text("Back"), button:has-text("Volver")').isVisible().catch(() => false);
    console.log(`   Back to login visible: ${backToLoginVisible}`);
    
    // Check if Send Code button exists
    const sendCodeVisible = await page.locator('button:has-text("Send Code"), button:has-text("Enviar Código")').isVisible().catch(() => false);
    console.log(`   Send Code button visible: ${sendCodeVisible}`);
    
    // Check if username input for forgot flow exists
    const forgotInputVisible = await page.locator('input[placeholder*="user"]').count();
    console.log(`   Username inputs count: ${forgotInputVisible}`);

    // Try clicking with Playwright locator directly
    console.log('\n5. Trying direct Playwright click...');
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    
    const forgotLink = page.locator('text=Forgot password?').first();
    if (await forgotLink.isVisible()) {
      console.log('   Found link, clicking...');
      await forgotLink.click({ force: true });
      await page.waitForTimeout(2000);
      
      await page.screenshot({ path: 'test-screenshots/simple-03-playwright-click.png' });
      
      const buttonsAfter = await page.evaluate(() => {
        return [...document.querySelectorAll('button')].map(b => b.textContent.trim());
      });
      console.log(`   Buttons after: ${JSON.stringify(buttonsAfter)}`);
    }

  } catch (error) {
    console.error('\n💥 ERROR:', error.message);
  } finally {
    console.log('\n⏳ Browser open for 15s...');
    await page.waitForTimeout(15000);
    await browser.close();
  }
}

simpleForgotTest().catch(console.error);
