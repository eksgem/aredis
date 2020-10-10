### Description
The C library function char *setlocale(int category, const char *locale) sets or reads location dependent information.See https://www.tutorialspoint.com/c_standard_library/c_function_setlocale.htm

### Declaration
char *setlocale(int category, const char *locale)
#### Parameters
- category − This is a named constant specifying the category of the functions affected by the locale setting.

    LC_ALL for all of the below.

    LC_COLLATE for string comparison. See strcoll().

    LC_CTYPE for character classification and conversion. For example − strtoupper().

    LC_MONETARY for monetary formatting for localeconv().

    LC_NUMERIC for decimal separator for localeconv().

    LC_TIME for date and time formatting with strftime().

    LC_MESSAGES for system responses.

- locale − If locale is NULL or the empty string "", the locale names will be set from the values of environment variables with the same names as the above categories.

#### Return Value
A successful call to setlocale() returns an opaque string that corresponds to the locale set. The return value is NULL if the request cannot be honored.


