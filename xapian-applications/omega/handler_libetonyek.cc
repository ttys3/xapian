/** @file
 * @brief Extract text and metadata from Apple documents using libtonyek.
 */
/* Copyright (C) 2019 Bruno Baruffaldi
 * Copyright (C) 2022 Olly Betts
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation; either version 2 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA  02110-1301
 * USA
 */
#include <config.h>
#include "handler.h"
#include "stringutils.h"

#include <memory>

#include <librevenge-generators/librevenge-generators.h>
#include <librevenge-stream/librevenge-stream.h>
#include <libetonyek/libetonyek.h>

#define PARSE_FIELD(START, END, FIELD, OUT...) \
   parse_metadata_field((START), (END), (FIELD), (CONST_STRLEN(FIELD)), OUT)

using libetonyek::EtonyekDocument;
using namespace librevenge;
using namespace std;

// Handle a field for which we only take a single value - we avoid copying in
// this case.
static void
parse_metadata_field(const char* start,
		     const char* end,
		     const char* field,
		     size_t len,
		     const char*& out,
		     size_t& out_len)
{
    if (size_t(end - start) > len && memcmp(start, field, len) == 0) {
	start += len;
	while (start != end && isspace(*start)) start++;
	if (start != end && (end[-1] != '\r' || --end != start)) {
	    out = start;
	    out_len = end - start;
	}
    }
}

// Handle a field for which we concatenate multiple instances.  We need to copy
// in this case.
static void
parse_metadata_field(const char* start,
		     const char* end,
		     const char* field,
		     size_t len,
		     string& out)
{
    if (size_t(end - start) > len && memcmp(start, field, len) == 0) {
	start += len;
	while (start != end && isspace(*start)) start++;
	if (start != end && (end[-1] != '\r' || --end != start)) {
	    if (!out.empty())
		out.push_back(' ');
	    out.append(start, end - start);
	}
    }
}

static void
parse_metadata(const char* data, size_t len,
	       const char*& title, size_t& title_len,
	       const char*& author, size_t& author_len,
	       string& keywords)
{
    const char* p = data;
    const char* end = p + len;

    while (p != end) {
	const char* start = p;
	p = static_cast<const char*>(memchr(p, '\n', end - start));
	const char* eol;
	if (p)
	    eol = p++;
	else
	    p = eol = end;
	if ((end - start) > 5 && memcmp(start, "meta:", 5) == 0) {
	    start += 5;
	    switch (*start) {
		case 'i': {
		    // Use dc:creator in preference to meta:initial-creator.
		    if (!author)
			PARSE_FIELD(start, eol, "initial-creator",
				    author, author_len);
		    break;
		}
		case 'k': {
		    PARSE_FIELD(start, eol, "keyword", keywords);
		    break;
		}
	    }
	} else if ((end - start) > 3 && memcmp(start, "dc:", 3) == 0) {
	    start += 3;
	    switch (*start) {
		case 'c': {
		    // Use dc:creator in preference to meta:initial-creator.
		    PARSE_FIELD(start, eol, "creator", author, author_len);
		    break;
		}
		case 's': {
		    PARSE_FIELD(start, eol, "subject", keywords);
		    break;
		}
		case 't': {
		    PARSE_FIELD(start, eol, "title", title, title_len);
		    break;
		}
	    }
	} else if ((end - start) > 8 && memcmp(start, "dcterms:", 8) == 0) {
	    start += 8;
	    PARSE_FIELD(start, eol, "available", keywords);
	}
    }
}

static void
extract_key(RVNGInputStream* input)
{
    RVNGStringVector content;
    RVNGTextPresentationGenerator document(content);
    if (!EtonyekDocument::parse(input, &document)) {
	fail("Libetonyek Error: Fail to extract text");
	return;
    }
    string dump;
    unsigned size = content.size();
    for (unsigned i = 0; i < size; ++i) {
	if (i) dump += '\n';
	dump.append(content[i].cstr(), content[i].size());
    }
    response(dump.data(), dump.size(),
	     nullptr, 0,
	     nullptr, 0,
	     nullptr, 0,
	     -1);
}

static void
extract_numbers(RVNGInputStream* input)
{
    RVNGStringVector content;
    RVNGTextSpreadsheetGenerator document(content);

    if (!EtonyekDocument::parse(input, &document)) {
	fail("Libetonyek Error: Fail to extract text");
	return;
    }
    string dump;
    unsigned size = content.size();
    for (unsigned i = 0; i < size; ++i) {
	if (i) dump += '\n';
	dump.append(content[i].cstr(), content[i].size());
    }
    response(dump.data(), dump.size(),
	     nullptr, 0,
	     nullptr, 0,
	     nullptr, 0,
	     -1);
}

static void
extract_pages(RVNGInputStream* input)
{
    RVNGString dump, metadata;

    // Extract metadata.
    RVNGTextTextGenerator data(metadata, true);
    if (!EtonyekDocument::parse(input, &data)) {
	fail("Libetonyek Error: Fail to extract metadata");
	return;
    }

    const char* title = nullptr;
    const char* author = nullptr;
    size_t title_len = 0, author_len = 0;
    string keywords;
    parse_metadata(metadata.cstr(), metadata.size(),
		   title, title_len,
		   author, author_len,
		   keywords);

    // Extract body text.
    RVNGTextTextGenerator content(dump, false);
    if (!EtonyekDocument::parse(input, &content)) {
	fail("Libetonyek Error: Fail to extract text");
	return;
    }

    response(dump.cstr(), dump.size(),
	     title, title_len,
	     keywords.data(), keywords.size(),
	     author, author_len,
	     -1);
}

void
extract(const string& filename,
	const string& mimetype)
try {
    unique_ptr<RVNGInputStream> input;

    if (RVNGDirectoryStream::isDirectory(filename.c_str()))
	input.reset(new RVNGDirectoryStream(filename.c_str()));
    else
	input.reset(new RVNGFileStream(filename.c_str()));

    EtonyekDocument::Type type = EtonyekDocument::TYPE_UNKNOWN;
    auto confidence = EtonyekDocument::isSupported(input.get(), &type);

    if (confidence == EtonyekDocument::CONFIDENCE_NONE) {
	fail("Libetonyek Error: The format couldn't be detected");
	return;
    }

    switch (type) {
	case EtonyekDocument::TYPE_PAGES:
	    extract_pages(input.get());
	    return;
	case EtonyekDocument::TYPE_NUMBERS:
	    extract_numbers(input.get());
	    return;
	case EtonyekDocument::TYPE_KEYNOTE:
	    extract_key(input.get());
	    return;
	default:
	    fail("Libetonyek Error: The format is not supported");
    }
} catch (...) {
    fail("Libetonyek threw an exception");
}
