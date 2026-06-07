let web3;
let contract;
let userAccount;
let contractAddress = "0x4b6f8cC633F39eCBf5c959B2c9B5F570D8D47CD1"; 

// Функція додавання повідомлень у лог
function logEvent(message) {
    const logDiv = document.getElementById("eventLog");
    const time = new Date().toLocaleTimeString();
    logDiv.innerHTML += `<div>[${time}] ${message}</div>`;
    logDiv.scrollTop = logDiv.scrollHeight;
}

// Підключення до MetaMask та ініціалізація
async function connectWallet() {
    if (typeof window.ethereum === 'undefined') {
        alert("MetaMask not installed");
        return;
    }
    try {
        const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
        userAccount = accounts[0];
        web3 = new Web3(window.ethereum);
        
        // Ensure we are on Sepolia
        await ensureSepoliaNetwork();
        document.getElementById("networkStatus").innerHTML = "🌐 Мережа: Sepolia";
        
        // Load contract...
        const response = await fetch('/static/abi.json');
        const abi = await response.json();
        contract = new web3.eth.Contract(abi, contractAddress);
        
        // Check balance on Sepolia
        const balanceWei = await web3.eth.getBalance(userAccount);
        const balanceEth = web3.utils.fromWei(balanceWei, 'ether');
        document.getElementById("status").innerHTML = `✅ Підключено: ${userAccount.slice(0,6)}... (Sepolia, баланс: ${balanceEth} ETH)`;
        
        if (parseFloat(balanceEth) === 0) {
            logEvent("⚠️ Увага: на цьому акаунті 0 ETH. Отримайте тестові ETH з крана.");
        }
        
        // Subscribe to events
        subscribeToEvents();
    } catch (error) {
        console.error(error);
        alert("Помилка: " + error.message);
    }
}

// Отримання балансу (read)
async function getBalance(address) {
    if (!web3) { alert("Підключіть гаманець"); return; }
    if (!web3.utils.isAddress(address)) { alert("Некоректна адреса"); return; }
    try {
        const balanceWei = await web3.eth.getBalance(address);
        const balanceEth = web3.utils.fromWei(balanceWei, 'ether');
        document.getElementById("balanceDisplay").innerHTML = `${balanceEth} ETH (нативний)`;
        logEvent(`Нативний баланс ${address}: ${balanceEth} ETH`);
    } catch (err) {
        alert("Помилка: " + err.message);
    }
}

// Депозит (запис – транзакція)
async function deposit() {
    if (!contract || !userAccount) { alert("Підключіть гаманець"); return; }
    const amountEth = document.getElementById("depositAmount").value;
    if (!amountEth || amountEth <= 0) { alert("Введіть коректну суму"); return; }
    const amountWei = web3.utils.toWei(amountEth, 'ether');
    try {
        const tx = await contract.methods.deposit().send({
            from: userAccount,
            value: amountWei,
            gas: 300000
        });
        logEvent(`✅ Депозит ${amountEth} ETH. Tx: ${tx.transactionHash.slice(0,10)}...`);
        // Оновити баланс для поточного користувача
        await getBalance(userAccount);
    } catch (err) {
        alert("Помилка депозиту: " + err.message);
    }
}

// Переказ (транзакція)
async function transfer() {
    if (!contract || !userAccount) { alert("Підключіть гаманець"); return; }
    const toAddress = document.getElementById("transferTo").value.trim();
    const amountEth = document.getElementById("transferAmount").value;
    if (!web3.utils.isAddress(toAddress)) { alert("Некоректна адреса отримувача"); return; }
    if (!amountEth || amountEth <= 0) { alert("Введіть суму"); return; }
    const amountWei = web3.utils.toWei(amountEth, 'ether');
    try {
        const tx = await contract.methods.transfer(toAddress, amountWei).send({
            from: userAccount,
            gas: 300000
        });
        logEvent(`🔄 Переказ ${amountEth} ETH на ${toAddress.slice(0,6)}... Tx: ${tx.transactionHash.slice(0,10)}...`);
        await getBalance(userAccount);
    } catch (err) {
        alert("Помилка переказу: " + err.message);
    }
}

// Підписка на події (Deposited, Transferred) – реальний час
function subscribeToEvents() {
    if (!contract) return;
    
    // Подія Deposited
    contract.events.Deposited({
        fromBlock: 'latest'
    }, (error, event) => {
        if (error) console.error(error);
        else {
            const user = event.returnValues.user;
            const amountEth = web3.utils.fromWei(event.returnValues.amount, 'ether');
            logEvent(`💰 Подія Deposited: ${user.slice(0,6)}... поповнив на ${amountEth} ETH`);
        }
    });
    
    // Подія Transferred
    contract.events.Transferred({
        fromBlock: 'latest'
    }, (error, event) => {
        if (error) console.error(error);
        else {
            const from = event.returnValues.from;
            const to = event.returnValues.to;
            const amountEth = web3.utils.fromWei(event.returnValues.amount, 'ether');
            logEvent(`🔄 Подія Transferred: ${from.slice(0,6)}... -> ${to.slice(0,6)}..., сума ${amountEth} ETH`);
        }
    });
    
    logEvent("Підписку на події активовано");
}

// Зв'язування кнопок
window.onload = () => {
    document.getElementById("connectBtn").onclick = connectWallet;
    document.getElementById("getBalanceBtn").onclick = () => {
        let addr = document.getElementById("balanceAddress").value.trim();
        if (!addr) addr = userAccount;
        if (!addr) { alert("Спочатку підключіть гаманець або введіть адресу"); return; }
        getBalance(addr);
    };
    document.getElementById("depositBtn").onclick = deposit;
    document.getElementById("transferBtn").onclick = transfer;
};

async function ensureSepoliaNetwork() {
    const chainId = await web3.eth.getChainId();
    if (chainId !== 11155111) { // Sepolia chain ID
        try {
            await window.ethereum.request({
                method: 'wallet_switchEthereumChain',
                params: [{ chainId: '0xaa36a7' }], // 11155111 in hex
            });
        } catch (err) {
            if (err.code === 4902) {
                alert('Sepolia not added. Please add it manually in MetaMask.');
            }
            throw err;
        }
    }
}