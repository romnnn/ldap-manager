package hash

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	// "crypto/des"
	"crypto/sha1"
	// "math/big"
	"github.com/GehirnInc/crypt"
	// UNIX crypt(3)
	_ "github.com/GehirnInc/crypt/md5_crypt"
	_ "github.com/GehirnInc/crypt/sha256_crypt"
	_ "github.com/GehirnInc/crypt/sha512_crypt"
)

// LDAPPasswordHashingAlgorithm ...
type LDAPPasswordHashingAlgorithm uint16

const (
	// Default ...
	Default LDAPPasswordHashingAlgorithm = iota
	// SHA512CRYPT ...
	SHA512CRYPT
	// SHA256CRYPT ...
	SHA256CRYPT
	// BLOWFISH ...
	BLOWFISH
	// EXTDES ...
	EXTDES
	// MD5CRYPT ...
	MD5CRYPT
	// SMD5 ...
	SMD5
	// MD5 ...
	MD5
	// SHA ...
	SHA
	// SSHA ...
	SSHA
	// CRYPT ...
	CRYPT
	// CLEAR ...
	CLEAR
)

// LDAPPasswordHashingAlgorithms ...
var LDAPPasswordHashingAlgorithms = map[string]LDAPPasswordHashingAlgorithm{
	"Default":     Default,
	"SHA512CRYPT": SHA512CRYPT,
	"SHA256CRYPT": SHA256CRYPT,
	// "BLOWFISH":    BLOWFISH,
	// "EXTDES":      EXTDES,
	"MD5CRYPT": MD5CRYPT,
	"SMD5":     SMD5,
	"MD5":      MD5,
	"SHA":      SHA,
	"SSHA":     SSHA,
	// "CRYPT":       CRYPT,
	"CLEAR": CLEAR,
}

func encodeSSHA(pw string) string {
	// $salt = generate_salt(8)
	// '{SSHA}' . base64_encode(sha1($password . $salt, TRUE) . $salt)
	sha := sha1.New()
	salt := generateSalt(4)
	sha.Write([]byte(pw))
	sha.Write(salt)
	hash := append(sha.Sum(nil), salt...)
	return fmt.Sprintf("{SSHA}%s", base64.StdEncoding.EncodeToString(hash))
}

func encodeSMD5(pw string) string {
	// $salt = generate_salt(8)
	// '{SMD5}' . base64_encode(md5($password . $salt, TRUE) . $salt)
	sha := md5.New()
	salt := generateSalt(4)
	sha.Write([]byte(pw))
	sha.Write(salt)
	hash := append(sha.Sum(nil), salt...)
	return fmt.Sprintf("{SMD5}%s", base64.StdEncoding.EncodeToString(hash))
}

func encodeSHA256(pw string) string {
	// '{CRYPT}' . crypt($password, '$5$' . generate_salt(8))
	crypt := crypt.SHA256.New()
	hash, _ := crypt.Generate([]byte(pw), append([]byte("$5$"), generateSalt(8)...))
	return fmt.Sprintf("{CRYPT}%s", hash)
}

func encodeSHA512(pw string) string {
	// '{CRYPT}' . crypt($password, '$6$' . generate_salt(8));
	crypt := crypt.SHA512.New()
	hash, _ := crypt.Generate([]byte(pw), append([]byte("$6$"), generateSalt(8)...))
	return fmt.Sprintf("{CRYPT}%s", hash)
}

func encodeMD5CRYPT(pw string) string {
	// '{CRYPT}' . crypt($password, '$1$' . generate_salt(9));
	crypt := crypt.MD5.New()
	hash, _ := crypt.Generate([]byte(pw), append([]byte("$1$"), generateSalt(9)...))
	return fmt.Sprintf("{CRYPT}%s", hash)
}

func encodeCLEAR(pw string) string {
	return pw
}

func encodeMD5(pw string) string {
	// '{MD5}' . base64_encode(md5($password, TRUE));
	h := md5.New()
	h.Write([]byte(pw))
	return fmt.Sprintf("{MD5}%s", base64.StdEncoding.EncodeToString(h.Sum(nil)))
}

func encodeSHA(pw string) string {
	// '{SHA}' . base64_encode(sha1($password, TRUE));
	h := sha1.New()
	h.Write([]byte(pw))
	return fmt.Sprintf("{SHA}%s", base64.StdEncoding.EncodeToString(h.Sum(nil)))
}

func encodeBLOWFISH(pw string) string {
	// '{CRYPT}' . crypt($password, '$2a$12$' . generate_salt(13));
	hash, _ := bcrypt([]byte(pw), 13, generateSalt(13))
	fmt.Println(string(hash))
	return fmt.Sprintf("{CRYPT}%s", string(hash))
}

func encodeCRYPT(pw string) string {
	/*
		The checksum is formed by a modified version of the DES cipher in encrypt mode:

		Given a password string and a salt string.

		The 2 character salt string is decoded to a 12-bit integer salt value; The salt string uses little-endian hash64 encoding.

		If the password is less than 8 bytes, it’s NULL padded at the end to 8 bytes.

		The lower 7 bits of the first 8 bytes of the password are used to form a 56-bit integer; with the first byte providing the most significant 7 bits, and the 8th byte providing the least significant 7 bits.

		The remainder of the password (if any) is ignored.

		25 repeated rounds of modified DES encryption are performed; starting with a null input block, and using the 56-bit integer from step 4 as the DES key.

		The salt is used to to mutate the normal DES encrypt operation by swapping bits i and i+24 in the DES E-Box output if and only if bit i is set in the salt value. Thus, if the salt is set to 0, normal DES encryption is performed. (This was intended to prevent optimized implementations of regular DES encryption to be useful in attacking this algorithm).

		The 64-bit result of the last round of step 5 is then lsb-padded with 2 zero bits.

		The resulting 66-bit integer is encoded in big-endian order using the hash64-big format.
	*/

	// '{CRYPT}' . crypt($password, generate_salt(2));
	// here, crypt uses CRYPT_STD_DES
	// c, _ := des.NewCipher(append([]byte(pw), generateSalt(2)...))

	/*
		salt := "JQ"
		h := md5.New()
		h.Write([]byte(pw))
	*/
	hashed := Crypt(pw, "JQ")
	return fmt.Sprintf("{CRYPT}%s", hashed)
	// return fmt.Sprintf("{CRYPT}%s", )
}

// generateSalt generates a byte array containing random bytes
func generateSalt(l int) []byte {
	sbytes := make([]byte, l)
	rand.Read(sbytes)
	return sbytes
}

// Password ...
func Password(password string, algorithm LDAPPasswordHashingAlgorithm) (string, error) {
	switch algorithm {
	case SSHA, Default:
		return encodeSSHA(password), nil
	case SHA256CRYPT:
		return encodeSHA256(password), nil
	case SHA512CRYPT:
		return encodeSHA512(password), nil
	case BLOWFISH:
		return "", errors.New("BLOWFISH is currently not supported")
		// return encodeBLOWFISH(password), nil
	case MD5:
		return encodeMD5(password), nil
	case MD5CRYPT:
		return encodeMD5CRYPT(password), nil
	case SMD5:
		return encodeSMD5(password), nil
	case SHA:
		return encodeSHA(password), nil
	case CRYPT:
		return "", errors.New("CRYPT is currently not supported")
		// return encodeCRYPT(password), nil
	case EXTDES:
		return "", errors.New("EXTDES is currently not supported")
		// return encodeEXTDES(password), nil
	case CLEAR:
		return encodeCLEAR(password), nil
	default:
		return encodeSSHA(password), nil
	}
}

/*
$hash_algo = 'SSHA';

	switch ($hash_algo) {

   # Blowfish & EXT_DES didn't work
   #  case 'BLOWFISH':
   #    $hashed_pwd = '{CRYPT}' . crypt($password, '$2a$12$' . generate_salt(13));
   #    break;

   #  case 'EXT_DES':
   #    $hashed_pwd = '{CRYPT}' . crypt($password, '_' . generate_salt(8));
   #    break;

	 case 'MD5CRYPT':
	   $hashed_pwd = '{CRYPT}' . crypt($password, '$1$' . generate_salt(9));
	   break;

	 case 'SMD5':
	   $salt = generate_salt(8);
	   $hashed_pwd = '{SMD5}' . base64_encode(md5($password . $salt, TRUE) . $salt);
	   break;

	 case 'SHA':
	   $hashed_pwd = '{SHA}' . base64_encode(sha1($password, TRUE));
	   break;

	 case 'SSHA':
	   $salt = generate_salt(8);
	   $hashed_pwd = '{SSHA}' . base64_encode(sha1($password . $salt, TRUE) . $salt);
	   break;

	 case 'CRYPT':
	   $salt = generate_salt(2);
	   $hashed_pwd = '{CRYPT}' . crypt($password, $salt);
	   break;

	 case 'CLEAR':
	   error_log("$log_prefix password hashing - WARNING - Saving password in cleartext. This is extremely bad practice and should never ever be done in a production environment.");
	   $hashed_pwd = $password;
	   break;


	}
*/

/*
function ldap_hashed_password($password) {

	global $PASSWORD_HASH, $log_prefix;

	$check_algos = array (
						  "SHA512CRYPT" => "CRYPT_SHA512",
						  "SHA256CRYPT" => "CRYPT_SHA256",
   #                       "BLOWFISH"    => "CRYPT_BLOWFISH",
   #                       "EXT_DES"     => "CRYPT_EXT_DES",
						  "MD5CRYPT"    => "CRYPT_MD5"
						 );

	$remaining_algos = array (
							   "SSHA",
							   "SHA",
							   "SMD5",
							   "MD5",
							   "CRYPT",
							   "CLEAR"
							 );

	$available_algos = array();

	foreach ($check_algos as $algo_name => $algo_function) {
	  if (defined($algo_function) and constant($algo_function) != 0) {
		array_push($available_algos, $algo_name);
	  }
	  else {
		error_log("$log_prefix password hashing - the system doesn't support ${algo_name}");
	  }
	}
	$available_algos = array_merge($available_algos, $remaining_algos);

	if (isset($PASSWORD_HASH)) {
	  if (!in_array($PASSWORD_HASH, $available_algos)) {
		$hash_algo = $available_algos[0];
		error_log("$log_prefix LDAP password: the chosen hash method ($PASSWORD_HASH) wasn't available");
	  }
	  else {
		$hash_algo = $PASSWORD_HASH;
	  }
	}
	else {
	  $hash_algo = $available_algos[0];
	}
	error_log("$log_prefix LDAP password: using '${hash_algo}' as the hashing method");

	$hash_algo = 'SSHA';

	switch ($hash_algo) {

	 case 'SHA512CRYPT':
	   $hashed_pwd = '{CRYPT}' . crypt($password, '$6$' . generate_salt(8));
	   break;

	 case 'SHA256CRYPT':
	   $hashed_pwd = '{CRYPT}' . crypt($password, '$5$' . generate_salt(8));
	   break;

   # Blowfish & EXT_DES didn't work
   #  case 'BLOWFISH':
   #    $hashed_pwd = '{CRYPT}' . crypt($password, '$2a$12$' . generate_salt(13));
   #    break;

   #  case 'EXT_DES':
   #    $hashed_pwd = '{CRYPT}' . crypt($password, '_' . generate_salt(8));
   #    break;

	 case 'MD5CRYPT':
	   $hashed_pwd = '{CRYPT}' . crypt($password, '$1$' . generate_salt(9));
	   break;

	 case 'SMD5':
	   $salt = generate_salt(8);
	   $hashed_pwd = '{SMD5}' . base64_encode(md5($password . $salt, TRUE) . $salt);
	   break;

	 case 'MD5':
	   $hashed_pwd = '{MD5}' . base64_encode(md5($password, TRUE));
	   break;

	 case 'SHA':
	   $hashed_pwd = '{SHA}' . base64_encode(sha1($password, TRUE));
	   break;

	 case 'SSHA':
	   $salt = generate_salt(8);
	   $hashed_pwd = '{SSHA}' . base64_encode(sha1($password . $salt, TRUE) . $salt);
	   break;

	 case 'CRYPT':
	   $salt = generate_salt(2);
	   $hashed_pwd = '{CRYPT}' . crypt($password, $salt);
	   break;

	 case 'CLEAR':
	   error_log("$log_prefix password hashing - WARNING - Saving password in cleartext. This is extremely bad practice and should never ever be done in a production environment.");
	   $hashed_pwd = $password;
	   break;


	}

	error_log("$log_prefix password update - algo $hash_algo | pwd $hashed_pwd");

	return $hashed_pwd;

   }
*/
