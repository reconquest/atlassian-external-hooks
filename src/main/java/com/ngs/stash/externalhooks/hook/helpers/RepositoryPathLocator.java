package com.ngs.stash.externalhooks.hook.helpers;

import com.atlassian.bitbucket.repository.Repository;
import com.atlassian.bitbucket.server.StorageService;
import com.atlassian.cache.Cache;
import com.atlassian.cache.CacheFactory;
import org.timo.gitconfig.Configuration;
import org.timo.gitconfig.GitConfiguration;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Optional;
import java.util.logging.Logger;

import static java.util.logging.Level.SEVERE;

public class RepositoryPathLocator {

    public static final String REPOSITORY_CONFIG_FILE = "repository-config";
    private static Logger log = Logger.getLogger(
            RepositoryPathLocator.class.getSimpleName()
    );

    private final Cache<Integer, String> cache;

    public RepositoryPathLocator(CacheFactory cacheFactory) {
        this.cache = cacheFactory.getCache("com.ngs.stash.externalhooks.external-hooks:repo-paths-caches");
    }

    /**
     * Take our best guess at what the folder for the repo is
     *
     * @param repository the repo we are trying to guess the path to
     * @param storageService service that provides home dir config
     * @return Path to the repo, or <code>null</code> if we cannot find one
     */
    public Path getRepositoryDir(Repository repository, StorageService storageService) {
        Optional<Path> maybePath = getCachedPath(repository);
        if (maybePath.isPresent()) {
            return maybePath.get();
        }

        String basePath = storageService.getSharedHomeDir() + "/data";
        Path repoPath = guessRepoPath(repository, basePath);
        if (repoPath != null) {
            cache.put(repository.getId(), repoPath.toAbsolutePath().toString());
            return repoPath;
        } else {
            try {
                maybePath = Files.find(Paths.get(basePath),
                        Integer.MAX_VALUE,
                        (path, basicFileAttributes) -> path.endsWith(REPOSITORY_CONFIG_FILE) && basicFileAttributes.isRegularFile())
                        .filter(path -> matchesRepo(repository, path.getParent()))
                        .findFirst();
                if (maybePath.isPresent()) {
                    Path path = maybePath.get().getParent();
                    cache.put(repository.getId(), path.toAbsolutePath().toString());
                    return path;
                }
            } catch (IOException e) {
                log.log(SEVERE, "We were not able to search for the file due to exception", e);
            }
        }
        return null;
    }

    private Optional<Path> getCachedPath(Repository repository) {
        String cacheHit = cache.get(repository.getId());
        if (cacheHit != null) {
            Path cachePath = Paths.get(cacheHit);
            if (matchesRepo(repository, cachePath)) {
                return Optional.of(cachePath);
            }
        }
        return Optional.empty();
    }

    /** Guess repo path based on repo ID, check content of repository-config to confirm
     *
     * @param repository repo we are looking for
     * @param basePath path to start our search at
     * @return Path to the repo if the guess pinned out, <code>null</code> otherwise
     */
    private Path guessRepoPath(Repository repository, String basePath) {
        Path path = Paths.get(basePath, "repositories", String.valueOf(repository.getId()));
        if (Files.exists(path) && matchesRepo(repository, path)) {
            return path;
        } else {
            return null;
        }
    }

    /**
     * Check content of the config file to see if we have correct repo
     *
     * @param repository repo we are looking for
     * @param path path to repository-config
     * @return <code>true</code> if repo matches the metadata
     */
    private boolean matchesRepo(Repository repository, Path path) {
        final Configuration config = new GitConfiguration();
        try {
            config.load(path.resolve(REPOSITORY_CONFIG_FILE).toString());
            if (repository.getProject().getKey().equals(config.getValue("bitbucket.project")) &&
                    repository.getName().equals(config.getValue("bitbucket.repository"))) {
                return true;
            }
        } catch (IOException ignore) {}
        return false;
    }
}
