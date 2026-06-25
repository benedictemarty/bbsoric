#!/usr/bin/env python3
"""
driver.py — pilote le serveur BBS Oric (bbsd) par socket TCP.

Le serveur émet de l'« OASCII » : du texte ASCII entremêlé de codes d'attribut
Téletexte ($00-$1F, couleurs encre/fond, double hauteur…). Ce driver connecte un
socket, envoie des frappes (navigation par touche unique), lit chaque « écran »,
et le rend en texte lisible (les codes d'attribut deviennent des points médians).

Usage :
    # smoke flow complet (build le binaire si besoin via SKILL.md d'abord) :
    python3 .claude/skills/run-bbsoric/driver.py [host] [port]
        -> connecte, capture bannière+menu, navigue (1,2,3), quitte.
           Écrit les captures texte dans /tmp/bbs-*.txt et imprime un résumé.
           Code de sortie 0 si les écrans attendus apparaissent, 1 sinon.

    # pilotage manuel depuis du code :
    from driver import BBS
    b = BBS("127.0.0.1", 6502); b.connect()
    print(b.render(b.read_screen()))   # bannière + menu
    b.send("1"); print(b.render(b.read_screen()))
    b.close()
"""
import socket
import sys
import time


class BBS:
    def __init__(self, host="127.0.0.1", port=6502, timeout=4.0):
        self.host, self.port, self.timeout = host, port, timeout
        self.sock = None

    def connect(self):
        self.sock = socket.create_connection((self.host, self.port), self.timeout)
        self.sock.settimeout(0.6)

    def read_screen(self, settle=0.3, max_wait=4.0):
        """Lit un écran complet : attend l'arrivée des octets puis le silence."""
        buf = bytearray()
        deadline = time.time() + max_wait
        quiet = 0
        while time.time() < deadline:
            try:
                chunk = self.sock.recv(65536)
            except socket.timeout:
                # silence : on s'arrête seulement si on a déjà reçu quelque chose
                if buf:
                    quiet += 1
                    if quiet >= 1:
                        break
                continue
            if not chunk:
                break
            buf += chunk
            quiet = 0
            time.sleep(settle)
        return bytes(buf)

    def send(self, keys):
        self.sock.sendall(keys.encode("latin-1") if isinstance(keys, str) else keys)

    @staticmethod
    def render(data):
        """OASCII -> texte lisible (attributs $00-$1F -> '·', CR/LF -> saut)."""
        out = []
        for b in data:
            if b in (10, 13):
                out.append("\n")
            elif 32 <= b < 127:
                out.append(chr(b))
            else:
                out.append("·")
        # compacte les lignes vides multiples
        txt = "".join(out)
        return "\n".join(line.rstrip() for line in txt.splitlines())

    def close(self):
        if self.sock:
            try:
                self.sock.close()
            finally:
                self.sock = None


def main():
    host = sys.argv[1] if len(sys.argv) > 1 else "127.0.0.1"
    port = int(sys.argv[2]) if len(sys.argv) > 2 else 6502
    b = BBS(host, port)
    try:
        b.connect()
    except OSError as e:
        print(f"ECHEC : connexion {host}:{port} impossible ({e}).")
        print("Le serveur tourne-t-il ? cf. SKILL.md (./bbsd -addr ...).")
        return 1

    # Bannière + menu sur la première connexion déjà ouverte.
    banner = b.render(b.read_screen())
    open("/tmp/bbs-01-banner.txt", "w").write(banner)
    print("=== 01 bannière + menu principal ===")
    print(banner[:800])
    b.send("Q")
    time.sleep(0.2)
    b.close()

    checks = [("banniere+menu", "MENU PRINCIPAL" in banner and "BBS" in banner)]

    # Chaque écran de contenu est testé depuis une connexion FRAÎCHE :
    # déterministe (la nav après « Appuyez sur une touche » a un timing fragile).
    screens = [
        ("1", "02-info", "INFORMATIONS SYSTEME"),
        ("2", "03-apropos", "PROPOS"),
        ("3", "04-guestbook", "LIVRE"),
    ]
    for key, fname, expect in screens:
        c = BBS(host, port)
        c.connect()
        c.read_screen()       # bannière + menu
        c.send(key)
        scr = c.render(c.read_screen())
        c.send(" ")
        c.close()
        open(f"/tmp/bbs-{fname}.txt", "w").write(scr)
        checks.append((f"touche {key} -> {expect}", expect in scr))
        print(f"\n=== {fname} (touche {key}) ===")
        print(scr[:500])

    print("\n=== Résultat ===")
    ok = True
    for name, passed in checks:
        print(f"  [{'OK' if passed else 'FAIL'}] {name}")
        ok = ok and passed
    print("Captures : /tmp/bbs-01-banner.txt /tmp/bbs-02-info.txt"
          " /tmp/bbs-03-apropos.txt /tmp/bbs-04-guestbook.txt")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
