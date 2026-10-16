package ptcalculation

// NetCommission — commission สุทธิเป็น % ของยอดแทง (PTC-13 · แบบ ก: ชั้นบนตรงจ่าย commission ของลูกทั้งก้อน)
//
//	finalPT     = PT สุดท้ายของ L0 … Lk (ผลของ Calculate)
//	commissions = % Commission ที่ผู้สร้างตั้งให้ L0 … Lk (L0 ไม่ใช้)
//	memberCom   = % Commission ของ Member
//
// t[j] = Round4(%Com[j] × PT รวมของทุกชั้นเหนือ j ÷ 100) · Member ใช้ PT เหนือ = 100
// คืน L0 … Lk แล้วต่อด้วย Member (ยาว k + 2): L0 = −t[1] · Li = t[i] − t[i+1] · Member = t[Member] · รวมทุกค่า = 0
func NetCommission(finalPT, commissions []float64, memberCom float64) []float64 {
	n := len(finalPT)
	t := make([]float64, n+1) // t[0] ไม่ใช้ · t[n] = Member
	above := 0.0
	for j := 1; j <= n; j++ {
		above = round4(above + finalPT[j-1])
		com := memberCom
		if j < n {
			com = commissions[j]
		}
		t[j] = round4(com * above / 100)
	}

	net := make([]float64, n+1)
	for i := 0; i < n; i++ {
		net[i] = noNegZero(round4(t[i] - t[i+1]))
	}
	net[n] = noNegZero(t[n])
	return net
}

// noNegZero — 0 − 0 อาจได้ -0 ซึ่ง JSON เขียนเป็น "-0"
func noNegZero(v float64) float64 {
	if v == 0 {
		return 0
	}
	return v
}
