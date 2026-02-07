const { chromium } = require('playwright');

const BASE_URL = 'http://192.168.1.145:3002';

async function testForgotPasswordFlow() {
  console.log('🔐 Testing Complete Forgot Password Flow\n');
  
  const browser = await chromium.launch({ headless: false, slowMo: 400 });
  const context = await browser.newContext({ viewport: { width: 1200, height: 800 } });
  const page = await context.newPage();

  page.on('console', msg => {
    const text = msg.text();
    if (text.includes('[Login]') || text.includes('Forgot') || text.includes('error')) {
      console.log(`[Console]: ${text}`);
    }
  });

  try {
    console.log('1. Loading login page...');
    await page.goto(BASE_URL + '?t=' + Date.now(), { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    
    console.log('\n2. Clicking "Forgot Password?"...');
    const forgotBtn = page.locator('button').filter({ hasText: /Forgot|Olvidaste/ }).first();
    await forgotBtn.click();
    await page.waitForTimeout(1500);
    
    // List inputs
    const inputs = await page.evaluate(() => {
      return [...document.querySelectorAll('input')].map(i => ({ type: i.type, placeholder: i.placeholder, class: i.className.slice(0,50) }));
    });
    console.log('\n   Available inputs:', JSON.stringify(inputs, null, 2));
    
    console.log('\n3. Entering username (admin)...');
    const usernameInput = page.locator('input[type="text"]').first();
    await usernameInput.fill('admin');
    await page.waitForTimeout(500);
    await page.screenshot({ path: 'test-screenshots/forgot-2-request.png' });
    
    console.log('\n4. Clicking "Send Code"...');
    const sendBtn = page.locator('button').filter({ hasText: /Send|Enviar/ }).first();
    await sendBtn.click();
    await page.waitForTimeout(4000);
    await page.screenshot({ path: 'test-screenshots/forgot-3-after-send.png' });
    
    // Check what happened
    const pageState = await page.evaluate(() => {
      const buttons = [...document.querySelectorAll('button')].map(b => b.textContent.trim()).filter(t => t.length > 0 && t.length < 40);
      const inputs = [...document.querySelectorAll('input')].map(i => ({ type: i.type, placeholder: i.placeholder }));
      const errors = [...document.querySelectorAll('[class*="stopped"], [class*="error"], .text-red')].map(e => e.textContent.trim()).filter(t => t.length > 5);
      return { buttons, inputs, errors };
    });
    
    console.log('\n5. Page state after sending:');
    console.log('   Buttons:', pageState.buttons);
    console.log('   Inputs:', JSON.stringify(pageState.inputs));
    console.log('   Errors:', pageState.errors);
    
    // Check if we're on the code verification view (should have code input and password inputs)
    const hasVerifyView = await page.locator('text=/verification|verification code|código|Verify Code|Reset Password/i').count();
    console.log('\n   On verify view:', hasVerifyView > 0);

    console.log('\n✅ Test complete!');

  } catch (error) {
    console.error('\n💥 ERROR:', error.message);
    await page.screenshot({ path: 'test-screenshots/forgot-error.png' });
  } finally {
    console.log('\n⏳ Browser open for 8s...');
    await page.waitForTimeout(8000);
    await browser.close();
  }
}

testForgotPasswordFlow().catch(console.error);
